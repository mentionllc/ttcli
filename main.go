package main

import (
	"encoding/base64"
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
	ProductKey string `json:"product_key"`
	UploadUrl  string `json:"upload_url"`
}

func main() {
	var authFile string
	var configFile string
	// var eventType string

	app := cli.NewApp()
	app.Name = "ttcli"
	app.Usage = "for uploading data to Traintracks from the command line"
	app.Version = "0.0.1"
	app.Action = func(c *cli.Context) {
		send(authFile, configFile, c.Args()[0])
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "auth-file",
			Usage:       "file for authentication",
			Destination: &authFile,
		},
		cli.StringFlag{
			Name:        "config-file",
			Usage:       "config file with type data of file",
			Destination: &configFile,
		},
		// TODO: finish in next iteration
		// cli.StringFlag{
		// 	Name:        "event-type",
		// 	Usage:       "optional flag for explicitly using name for event type",
		// 	Destination: &eventType,
		// },
	}

	app.Run(os.Args)
}

func send(authFile string, configFile string, fileName string) {
	var err error
	var f *os.File
	var fi os.FileInfo
	var bar *pb.ProgressBar

	if f, err = os.Open(fileName); err != nil {
		log.Fatal(err)
	}
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

	var auth Auth
	b, err := ioutil.ReadFile(authFile)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(b, &auth)
	if err != nil {
		panic(err)
	}

	request, err := http.NewRequest("POST", auth.UploadUrl, r)
	if err != nil {
		log.Fatal(err)
	}

	b, err = ioutil.ReadFile(configFile)
	if err != nil {
		panic(err)
	}

	var configHash string
	configHash = base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(b)

	// TODO: finish in next iteration
	// if eventType == "" {
	// 	configHash = base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString(b)
	// } else {
	// 	// start eventType
	// 	var oldConfig map[string]interface{}
	// 	var newConfig map[string]interface{}
	// 	newConfig = map[string]interface{}{}
	// 	if err := json.Unmarshal(b, &oldConfig); err != nil {
	// 		panic(err)
	// 	}
	// 	i := 1
	// 	for k := range oldConfig {
	// 		name := fmt.Sprintf("%s%d", eventType, i)
	// 		newConfig[name+".csv"] = oldConfig[k]
	// 		i++
	// 	}
	// 	configJson, err := json.Marshal(newConfig)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	fmt.Print(newConfig)
	// 	configHash = base64.StdEncoding.WithPadding(base64.StdPadding).EncodeToString([]byte(configJson))
	// 	// end eventType
	// }

	request.Header.Set("Content-Type", mpw.FormDataContentType())
	request.Header.Set("QSV", "1")
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

	if resp.StatusCode == 200 {
		println("Data succcessfully sent.")
	} else {
		println("There was a problem with sending this file. Please email jeff@traintracks.io.")
		fmt.Print(string(ret))
	}
}
