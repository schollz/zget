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
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/schollz/httppool"
	log "github.com/schollz/logger"
	"github.com/schollz/progressbar/v3"
	"github.com/schollz/zget/src/httpstat"
	"github.com/schollz/zget/src/links"
	"github.com/schollz/zget/src/splicer"
	"github.com/schollz/zget/src/torrent"
	"github.com/schollz/zget/src/utils"
)

var flagWorkers int
var flagCompressed, flagVerbose, flagNoClobber, flagStdout, flagUseTor, flagDoStat, flagVersion, flagGzip, flagDownloadSite, flagDownloadSiteAll bool
var flagStripScript, flagStripStyle bool
var flagList, flagOutfile string
var flagHeaders arrayFlags
var Version = "v1.1.9-820d3b3"
var showTorIP bool
var spin *spinner.Spinner
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
	flag.StringVar(&flagOutfile, "o", "", "Filename to write download")
	flag.BoolVar(&flagStdout, "O", false, "Show in stdout")
	flag.IntVar(&flagWorkers, "w", 1, "Specify the number of workers")
	flag.Var(&flagHeaders, "H", "Pass custom header(s) to server")
	flag.BoolVar(&flagStripScript, "rm-script", false, "Remove script tags from downloaded HTML")
	flag.BoolVar(&flagStripStyle, "rm-style", false, "Remove style tags from download HTML")
	flag.BoolVar(&flagDownloadSiteAll, "site-all", false, "Download entire website and assets")
	flag.BoolVar(&flagDownloadSite, "site", false, "Download one website with assets")
}


func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `zget - like wget, but customized for zack
https://github.com/schollz/zget

USAGE:
  Download a webpage:
    zget -o nytimes.html nytimes.com

  Download a torrent:
    zget "magent:?...."
    zget "https://releases.ubuntu.com/.../ubuntu.torrent"

  Download a list of webpages, ignoring already downloaded:
    zget -nc -i urls.txt

  Download an entire site for uploading to IPFS:
    zget -w 10 --site-all -o $(date +%%F) schollz.com \
      ipfs add -r $(date +%%F) 

VERSION:
  `+Version+`

OPTIONS:
`)
		flag.VisitAll(func(f *flag.Flag) {
			s := fmt.Sprintf("  -%s", f.Name) // Two spaces before -; see next two comments.
			if len(strings.Fields(f.Name)[0]) > 1 {
				s = fmt.Sprintf("  --%s", f.Name) // Two spaces before -; see next two comments.
			}
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
		fmt.Printf("zget %s\n", Version)
		return
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

	if len(flag.Args()) > 0 {
		if strings.HasPrefix(flag.Args()[0], "magnet") || strings.HasSuffix(flag.Args()[0], ".torrent") {
			return torrent.Download(flag.Args()[0])
		}
		if flagDoStat {
			var uparsed *url.URL
			uparsed, err = utils.ParseURL(flag.Args()[0])
			if err != nil {
				return
			}
			httpstat.Run(uparsed, httpHeaders)
			os.Exit(0)
		}
	}

	hpool = httppool.New(
		httppool.OptionDebug(false),
		httppool.OptionNumClients(flagWorkers),
		httppool.OptionUseTor(flagUseTor),
		httppool.OptionHeaders(httpHeaders),
	)
	defer hpool.Close()
	if flagVerbose {
		log.SetLevel("debug")
	}

	if flagList != "" {
		err = downloadfromfile(flagList)
	} else if flagDownloadSite {
		err = downloadSite(flag.Args()[0], 2)
	} else if flagDownloadSiteAll {
		err = downloadSite(flag.Args()[0], 200)
	} else {
		_, _, _, _, err = download(flag.Args()[0], true, false)
	}

	return
}

func downloadSite(u string, depth int) (err error) {
	log.Trace(flagOutfile)
	if flagOutfile != "" {
		err = os.Chdir(flagOutfile)
		if err != nil {
			err = os.MkdirAll(flagOutfile, os.ModePerm)
			if err != nil {
				return
			}
			err = os.Chdir(flagOutfile)
			if err != nil {
				return
			}
		}
	}
	flagOutfile = ""

	flagNoClobber = true
	pagesToDo := make(map[string]struct{})
	pagesDone := make(map[string]struct{})

	uparsed, err := utils.ParseURL(u)
	if err != nil {
		return
	}
	pagesToDo[uparsed.String()] = struct{}{}

	log.Trace(depth)
	bar := progressbar.Default(1, "downloading "+uparsed.Host)
	bar.RenderBlank()
	iterations := -1
	for {
		iterations++
		if iterations == depth {
			break
		}
		linkstodo := make([]string, len(pagesToDo))
		i := 0
		for l := range pagesToDo {
			if _, ok := pagesDone[l]; !ok {
				linkstodo[i] = l
				i++
			}
		}
		if i == 0 {
			break
		}
		linkstodo = linkstodo[:i]
		bar.ChangeMax(bar.GetMax() + len(linkstodo))

		var numJobs = len(linkstodo)
		type job struct {
			u string
		}
		type result struct {
			u        string
			newlinks []string
			err      error
		}

		jobs := make(chan job, numJobs)
		results := make(chan result, numJobs)
		runtime.GOMAXPROCS(runtime.NumCPU())
		for i := 0; i < flagWorkers; i++ {
			go func(jobs <-chan job, results chan<- result) {
				for j := range jobs {
					var r result
					r.u = j.u
					var fpath string
					var ishtml bool
					_, fpath, _, ishtml, r.err = download(r.u, false, true)
					if r.err == nil && ishtml {
						var u2parsed *url.URL
						u2parsed, r.err = utils.ParseURL(r.u)
						if r.err == nil {
							if r.err == nil {
								r.newlinks, r.err = links.FromFile(fpath, u2parsed.String(), true, depth <= 3)
								if err != nil {
									r.newlinks, r.err = links.FromFile(path.Join(fpath, "index.html"), uparsed.String(), true, depth <= 3)
								}
							}
						}
					}
					results <- r
				}
			}(jobs, results)
		}

		for _, utodo := range linkstodo {
			jobs <- job{utodo}
		}
		close(jobs)

		for i := 0; i < numJobs; i++ {
			r := <-results
			bar.Add(1)
			pagesDone[r.u] = struct{}{}
			for _, newlink := range r.newlinks {
				pagesToDo[newlink] = struct{}{}
			}
		}

	}

	return
}

func downloadfromfile(fname string) (err error) {
	urlschan := make(chan string)
	done := make(chan bool)
	numDownloaded := 0
	bytesDownloaded := int64(0)
	var wg sync.WaitGroup
	wg.Add(flagWorkers)
	for i := 0; i < flagWorkers; i++ {
		go func() {
			defer wg.Done()
			log.Trace("starting worker")
			for {
				select {
				case <-done:
					log.Trace("exiting worker")
					return
				case u := <-urlschan:
					_, _, numBytesDownloaded, _, err := download(u, false, true)
					if numBytesDownloaded > 0 {
						numDownloaded++
					}
					bytesDownloaded += numBytesDownloaded
					if err != nil {
						log.Error(err)
					}
					log.Tracef("downloaded %s", u)
				}
			}

		}()
	}

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
	bar := progressbar.Default(
		int64(numLines),
	)
	bar.RenderBlank()
	for scanner.Scan() {
		bar.Add(1)
		u := strings.TrimSpace(scanner.Text())
		urlschan <- u
		// bar.Describe(u)
		// _, _, err = download(u, false, true)
		// if err != nil {
		// 	return
		// }
	}
	err = scanner.Err()

	for i := 0; i < flagWorkers; i++ {
		done <- true
	}
	wg.Wait()

	fmt.Fprintf(os.Stderr, "Downloaded %d URLs (%s)\n", numDownloaded, utils.HumanizeBytes(float64(bytesDownloaded)))
	return
}

func download(urlInput string, justone bool, indexhtml bool) (uget string, fpath string, nBytesDownloaded int64, ishtml bool, err error) {
	if justone {
		spin = spinner.New(spinner.CharSets[24], 100*time.Millisecond, spinner.WithWriter(os.Stderr))
		spin.Suffix = " connecting..."
		spin.Start()
		defer spin.Stop()
	}
	uparsed, err := utils.ParseURL(urlInput)
	if err != nil {
		return
	}

	uget = uparsed.String()
	fpath = path.Join(uparsed.Host, uparsed.Path)
	if strings.HasSuffix(uget, "/") {
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
			} else if stat.IsDir() && !indexhtml {
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

	resp, err := hpool.Get(uget)
	if err != nil {
		log.Trace(err)
		return
	}
	log.Tracef("resp: %+v", resp)
	if resp.StatusCode > 400 {
		err = fmt.Errorf(resp.Status)
		return
	}
	if flagGzip {
		fpath += ".gz"
	}

	if justone {
		spin.Stop()
		if !showTorIP && flagUseTor {
			showTorIP = !showTorIP
			ips, _ := hpool.PublicIP()
			if len(ips) > 0 {
				fmt.Fprintf(os.Stderr, "connected through tor as %s\n", ips[0])
			}
		}
	}
	defer resp.Body.Close()

	log.Tracef("Content-Type: %s", resp.Header.Get("Content-Type"))
	if indexhtml && strings.Contains(resp.Header.Get("Content-Type"), "html") && !strings.HasSuffix(fpath, ".html") && !strings.HasSuffix(fpath, ".htm") && !strings.HasSuffix(fpath, ".php") && !strings.HasSuffix(fpath, ".asp") {
		fpath = path.Join(fpath, "index.html")
	}
	foldername, _ := filepath.Split(fpath)
	log.Debugf("foldername: %s", foldername)
	os.MkdirAll(foldername, 0755)

	log.Tracef("fpath %s", fpath)
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}

	var writers []io.Writer

	var bar *progressbar.ProgressBar
	if justone && !flagStdout {
		bar = progressbar.DefaultBytes(
			resp.ContentLength,
			fpath,
		)
		bar.RenderBlank()
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
	} else if flagStdout {
		writers = append(writers, os.Stdout)
	} else {
		writers = append(writers, f)
	}
	dest := io.MultiWriter(writers...)
	nBytesDownloaded, err = io.Copy(dest, resp.Body)
	f.Close()
	if err != nil {
		return
	}

	// post processing`
	ishtml = strings.Contains(resp.Header.Get("Content-Type"), "html")
	if !flagGzip && ishtml {
		splicer.StripHTML(fpath, flagStripScript, flagStripStyle)
	}
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
