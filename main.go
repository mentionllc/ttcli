package main

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/codegangsta/cli"
)

type Auth struct {
	ProductKey    string `json:"product_key"`
	ProductSecret string `json:"product_secret"`
	CSVUrl        string `json:"csv_url"`
	EventsUrl     string `json:"events_url"`
}

func main() {
	var authFile string
	var configFile string
	var eventType string

	app := cli.NewApp()
	app.Name = "ttcli"
	app.Usage = "for uploading data to Traintracks from the command line"
	app.Action = func(c *cli.Context) {
		cli.ShowAppHelp(c)
	}
	app.Commands = []cli.Command{
		{
			Name:    "events",
			Aliases: []string{"e"},
			Usage:   "sends a file file of events",
			Action: func(c *cli.Context) {
				if !eventCheckFlags(authFile) {
					cli.ShowSubcommandHelp(c)
				} else {
					sendEvents(authFile, c.Args()[0])
				}
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "auth-file, a",
					Usage:       "file for authentication",
					Destination: &authFile,
				},
			},
		},
		{
			Name:  "csv",
			Usage: "sends a csv file",
			Action: func(c *cli.Context) {
				if !csvCheckFlags(authFile, configFile) {
					cli.ShowSubcommandHelp(c)
				} else {
					sendCSV(authFile, configFile, eventType, c.Args()[0])
				}
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "auth-file, a",
					Usage:       "file for authentication",
					Destination: &authFile,
				},
				cli.StringFlag{
					Name:        "config-file, c",
					Usage:       "config file with type information of file",
					Destination: &configFile,
				},
				cli.StringFlag{
					Name:        "event-type, et",
					Usage:       "optional flag for explicitly using name for event type",
					Destination: &eventType,
				},
			},
		},
	}
	app.Flags = []cli.Flag{}

	app.Run(os.Args)
}

func csvCheckFlags(auth string, config string) (res bool) {
	res = true
	if auth == "" {
		fmt.Println("You need to specify an auth file")
		res = false
	}
	if config == "" {
		fmt.Println("You need to specify a config file")
		res = false
	}
	return res
}

func eventCheckFlags(auth string) (res bool) {
	res = true
	if auth == "" {
		fmt.Println("You need to specify an auth file")
		res = false
	}
	return res
}

func sendCSV(authFile string, configFile string, eventType string, fileName string) {
	var err error
	var f *os.File
	var fi os.FileInfo
	var bar *pb.ProgressBar

	if f, err = os.Open(fileName); err != nil {
		log.Fatal(err)
	}

	// check file stats for progressbar
	if fi, err = f.Stat(); err != nil {
		log.Fatal(err)
	}
	bar = pb.New64(fi.Size()).SetUnits(pb.U_BYTES).SetRefreshRate(time.Millisecond * 10)
	bar.Start()

	r, w := io.Pipe()
	mpw := multipart.NewWriter(w)

	go func() {
		var part io.Writer
		defer w.Close()
		defer f.Close()

		if part, err = mpw.CreateFormFile("uploads", fi.Name()); err != nil {
			log.Fatal(err)
		}

		part = io.MultiWriter(part, bar)

		if _, err = io.Copy(part, f); err != nil {
			log.Fatal(err)
		}

		if err = mpw.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	// parse auth
	var auth Auth
	b, err := ioutil.ReadFile(authFile)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(b, &auth)
	if err != nil {
		panic(err)
	}

	request, err := http.NewRequest("POST", auth.CSVUrl, r)
	if err != nil {
		log.Fatal(err)
	}

	b, err = ioutil.ReadFile(configFile)
	if err != nil {
		panic(err)
	}

	configHash := base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(b)
	request.Header.Set("Content-Type", mpw.FormDataContentType())
	request.Header.Set("Configuration", configHash)
	request.Header.Set("X-Product-Key", auth.ProductKey)

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	ret, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(resp.StatusCode)
	fmt.Print(resp.Body)
	fmt.Print(string(ret))
}

func getProductAuth(key string, events []byte, secret string) string {
	hasher := md5.New()
	hasher.Write(append(events, []byte(secret)...))
	return hex.EncodeToString(hasher.Sum(nil))
}

func sendEvents(authFile string, fileName string) {

	// parse auth
	var auth Auth
	b, err := ioutil.ReadFile(authFile)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(b, &auth)
	if err != nil {
		panic(err)
	}

	f, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}

	file, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}

	productAuth := getProductAuth(auth.ProductKey, file, auth.ProductSecret)

	req, err := http.NewRequest("POST", auth.EventsUrl, bytes.NewBuffer(file))
	req.Header.Set("X-Product-Key", auth.ProductKey)
	req.Header.Set("X-Product-Auth", productAuth)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
}
