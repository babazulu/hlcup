package main

import (
	"flag"
	"log"
	"os"
	"runtime"
	"syscall"

	"github.com/valyala/fasthttp"
	"github.com/valyala/tcplisten"

	"github.com/ei-grad/hlcup/app"
	"github.com/ei-grad/hlcup/db"
)

var Version = "0.0.2"
var BuildDate string

func main() {

	log.Printf("HighLoad Cup solution by Andrew Grigorev <andrew@ei-grad.ru>")
	log.Printf("Version %s/DB=%s built %s, %s", Version, db.Version, BuildDate, runtime.Version())
	log.Printf("GOMAXPROCS: %d", runtime.GOMAXPROCS(0))

	var (
		accessLog     = flag.Bool("v", false, "show access log")
		address       = flag.String("b", ":80", "bind address")
		dataFileName  = flag.String("data", "/tmp/data/data.zip", "data file name")
		useHeat       = flag.Bool("heat", false, "heat GET requests on POST")
		runRpsWatcher = flag.Bool("rps", true, "log RPS every second")
	)

	flag.Parse()

	cpuinfo()
	swapon()
	rlimit()
	memstats()
	whoami()

	if os.Getenv("RUN_TOP") == "1" {
		go top()
	}

	app := app.NewApplication()
	app.UseHeat(*useHeat)
	if *runRpsWatcher {
		go app.RpsWatcher()
	}

	h := app.RequestHandler

	if *accessLog {
		h = accessLogHandler(h)
	}

	if err := syscall.Mlockall(syscall.MCL_CURRENT | syscall.MCL_FUTURE); err != nil {
		log.Print("mlockall: ", err)
	} else {
		log.Print("mlockall: success")
	}

	// goroutine to load data and profile cpu and mem
	go app.LoadData(*dataFileName)

	var err error

	var cfg = &tcplisten.Config{
		DeferAccept: true,
		FastOpen:    true,
	}
	ln, err := cfg.NewListener("tcp4", *address)
	if err != nil {
		log.Fatal("can't setup listener:", err)
	}
	if err := fasthttp.Serve(ln, h); err != nil {
		log.Fatal("fasthttp.Serve:", err)
	}
}
