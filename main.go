package main

//go:generate git tag -af v$VERSION -m "v$VERSION"
//go:generate go run .github/updateversion.go
//go:generate git commit -am "bump $VERSION"
//go:generate git tag -af v$VERSION -m "v$VERSION"

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/schollz/httppool"
	log "github.com/schollz/logger"
	"github.com/schollz/progressbar/v3"
	"github.com/schollz/zget/src/httpstat"
	"github.com/schollz/zget/src/torrent"
	"github.com/schollz/zget/src/utils"
)

var flagWorkers int
var flagCompressed, flagVerbose, flagNoClobber, flagUseTor, flagDoStat, flagVersion, flagGzip bool
var flagList, flagOutfile string
var flagHeaders arrayFlags
var Version = "v1.0.1-ffa5687"

var hpool *httppool.HTTPPool

func init() {
	flag.BoolVar(&flagCompressed, "compressed", false, "Request compressed response")
	flag.BoolVar(&flagVerbose, "v", false, "Verbosity mode")
	flag.BoolVar(&flagVersion, "version", false, "Print version")
	flag.BoolVar(&flagGzip, "gzip", false, "Download to gzipped file")
	flag.BoolVar(&flagNoClobber, "nc", false, "Skip downloads that are already retrieved")
	flag.BoolVar(&flagUseTor, "tor", false, "Use Tor proxy when downloading")
	flag.BoolVar(&flagDoStat, "stat", false, "Visualize curl statistics")
	flag.StringVar(&flagList, "i", "", "Download from a list of URLs")
	flag.StringVar(&flagOutfile, "o", "", "Filename to write download ")
	flag.IntVar(&flagWorkers, "w", 1, "Specify the number of workers")
	flag.Var(&flagHeaders, "H", "Pass custom header(s) to server")
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `zget - like wget, but customized for zack

USAGE:
	Download a webpage:
		zget -o nytimes.html nytimes.com

	Download a torrent:
		zget "magent:?...."
		zget "https://releases.ubuntu.com/.../ubuntu.torrent"

	Download a list of webpages, ignoring already downloaded:
		zget -nc -i urls.txt

VERSION:
	v`+Version+`

OPTIONS:
`)
		flag.VisitAll(func(f *flag.Flag) {
			s := fmt.Sprintf("  -%s", f.Name) // Two spaces before -; see next two comments.
			name, usage := flag.UnquoteUsage(f)
			if len(name) > 0 {
				s += " " + name
			}

			// Boolean flags of one ASCII letter are so common we
			// treat them specially, putting their usage on the same line.
			if len(s) <= 7 { // space, space, '-', 'x'.
				s += "\t\t"
			} else {
				// Four spaces before the tab triggers good alignment
				// for both 4- and 8-space tab stops.
				s += "\t"
			}

			s += strings.ReplaceAll(usage, "\n", "    \t")
			if !isZeroValue(f, f.DefValue) {
				if _, ok := f.Value.(*stringValue); ok {
					// put quotes on the value
					s += fmt.Sprintf(" (default %q)", f.DefValue)
				} else {
					s += fmt.Sprintf(" (default %v)", f.DefValue)
				}
			}
			fmt.Fprint(os.Stderr, s, "\n")
		})
	}
	flag.Parse()
	log.SetOutput(os.Stderr)
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s", err.Error())
		os.Exit(1)
	}
}

var httpHeaders map[string]string

func run() (err error) {
	if flagVersion {
		fmt.Printf("zget v%s\n", Version)
		return
	}
	if strings.HasPrefix(flag.Args()[0], "magnet") || strings.HasSuffix(flag.Args()[0], ".torrent") {
		return torrent.Download(flag.Args()[0])
	}
	if flagUseTor && runtime.GOOS == "windows" {
		err = fmt.Errorf("tor not supported on windows")
		return
	}
	httpHeaders = make(map[string]string)
	for _, header := range flagHeaders {
		foo := strings.SplitN(header, ":", 2)
		if len(foo) != 2 {
			continue
		}
		httpHeaders[strings.TrimSpace(foo[0])] = strings.TrimSpace(foo[1])
	}

	if flagDoStat {
		httpstat.Run(utils.ParseURL(flag.Args()[0]), httpHeaders)
		os.Exit(0)
	}
	hpool = httppool.New(
		httppool.OptionDebug(false),
		httppool.OptionNumClients(flagWorkers),
		httppool.OptionUseTor(flagUseTor),
		httppool.OptionHeaders(httpHeaders),
	)
	if flagVerbose {
		log.SetLevel("debug")
	}

	if flagList != "" {
		err = downloadfromfile(flagList)
	} else {
		err = download(flag.Args()[0], true)
	}

	return
}

func downloadfromfile(fname string) (err error) {
	numLines, err := countLinesInFile(fname)
	if err != nil {
		return
	}

	f, err := os.Open(fname)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	bar := progressbar.NewOptions(
		numLines,
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowIts(),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() { fmt.Println(" ") }),
		progressbar.OptionSetWidth(10),
	)
	for scanner.Scan() {
		bar.Add(1)
		u := strings.TrimSpace(scanner.Text())
		bar.Describe(u)
		err = download(u, false)
		if err != nil {
			return
		}
	}

	err = scanner.Err()
	return
}

func download(u string, justone bool) (err error) {
	uparsed := utils.ParseURL(u)
	u = uparsed.String()
	fpath := path.Join(uparsed.Host, uparsed.Path)
	if strings.HasSuffix(u, "/") {
		fpath = path.Join(fpath, "index.html")
	}
	log.Debugf("fpath: %s", fpath)
	if justone {
		_, filename := filepath.Split(fpath)
		fpath = filename
	}
	log.Debugf("fpath: %s", fpath)

	if flagOutfile != "" {
		fpath = flagOutfile
	} else {
		var stat os.FileInfo
		stat, err = os.Stat(fpath)
		if err == nil {
			if flagNoClobber {
				log.Debugf("already have %s", fpath)
				return
			} else if stat.IsDir() {
				err = fmt.Errorf("'%s' is directory: can't overwrite", fpath)
				return
			} else if !stat.IsDir() {
				for addNum := 1; addNum < 1000000; addNum++ {
					if _, errStat := os.Stat(fmt.Sprintf("%s.%d", fpath, addNum)); errStat != nil {
						fpath = fmt.Sprintf("%s.%d", fpath, addNum)
						break
					}
				}
			}
		}
	}
	if flagGzip {
		fpath += ".gz"
	}

	log.Debugf("saving to %s", fpath)
	resp, err := hpool.Get(u)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	foldername, _ := filepath.Split(fpath)
	log.Debugf("foldername: %s", foldername)
	os.MkdirAll(foldername, 0755)

	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	var writers []io.Writer

	var bar *progressbar.ProgressBar
	if justone {
		bar = progressbar.NewOptions(
			int(resp.ContentLength),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetDescription(fpath),
			progressbar.OptionOnCompletion(func() { fmt.Println(" ") }),
			progressbar.OptionSetWidth(10),
		)
		defer func() {
			bar.Finish()
		}()
		writers = append(writers, bar)
	}
	if flagGzip {
		buf := bufio.NewWriter(f)
		defer buf.Flush()
		gz := gzip.NewWriter(buf)
		defer gz.Close()
		writers = append(writers, gz)
	} else {
		writers = append(writers, f)
	}
	dest := io.MultiWriter(writers...)
	_, err = io.Copy(dest, resp.Body)
	return
}

func countLinesInFile(fname string) (int, error) {
	f, err := os.Open(fname)
	if err != nil {
		return -1, err
	}
	defer f.Close()
	return lineCounter(f)
}

func lineCounter(r io.Reader) (int, error) {
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}

// isZeroValue determines whether the string represents the zero
// value for a flag.
func isZeroValue(f *flag.Flag, value string) bool {
	// Build a zero value of the flag's Value type, and see if the
	// result of calling its String method equals the value passed in.
	// This works unless the Value type is itself an interface type.
	typ := reflect.TypeOf(f.Value)
	var z reflect.Value
	if typ.Kind() == reflect.Ptr {
		z = reflect.New(typ.Elem())
	} else {
		z = reflect.Zero(typ)
	}
	return value == z.Interface().(flag.Value).String()
}

// -- string Value
type stringValue string

func newStringValue(val string, p *string) *stringValue {
	*p = val
	return (*stringValue)(p)
}

func (s *stringValue) Set(val string) error {
	*s = stringValue(val)
	return nil
}

func (s *stringValue) Get() interface{} { return string(*s) }

func (s *stringValue) String() string { return string(*s) }

type arrayFlags []string

func (i *arrayFlags) String() string {
	return ""
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}
