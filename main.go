package main


import (
	"encoding/base64"
	"io/ioutil"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/howeyc/fsnotify"
	context "golang.org/x/net/context"
	speech "google.golang.org/api/speech/v1beta1"
	google "golang.org/x/oauth2/google"
)

var endpoint = "https://speech.googleapis.com/v1beta1/speech:syncrecognize"
var scope = speech.CloudPlatformScope
var bucketName = "willowbot"
var wd = ""

func main() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Print(err)
	}

	// Google Gubbins.
	client, err := google.DefaultClient(context.Background(), scope)
  if err != nil {
		log.Fatal(err)
  }

  speechSvc, err := speech.New(client)
  if err != nil {
		log.Fatal(err)
	}

	wd, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan bool)
	filesToProcess := make(chan string)

	go func() {
		for {
			select {
			case ev := <-watcher.Event:
				if !ev.IsCreate() {
					break
				}
				fmt.Println(fmt.Sprintf("listing file %v", ev.Name))
				filesToProcess <- ev.Name
				fmt.Println(ev.Name)
			case err :=<-watcher.Error:
				log.Print(err)
			}
		}
	}()

	err = watcher.Watch("output")
	if err != nil {
		log.Print(err)
	}

	// So basically we'll process these one at a time.
	for {
		toProcess := <-filesToProcess
		fmt.Println(fmt.Sprintf("Beginning to Process %v",toProcess))
		outcome := ProcessFile(toProcess, speechSvc)
		if outcome == nil {
			continue
		}
	}

	<-done
	watcher.Close()
}

var inProcessing = []string{}

func ProcessFile(fileName string, speechSvc *speech.Service) []string {
	// Check if we've already done this file.
	for _, f := range inProcessing {
		if f == fileName {
			return nil  // We've already done it
		}
	}
	cmdToRun := exec.Command("lsof", wd + "/" + fileName)
	out, err := cmdToRun.CombinedOutput()
	if err != nil {
		log.Print(err)
	}
	fmt.Println(out)

	for {}
  bytesThing, err := ioutil.ReadFile(wd + "/" + fileName)
  if err != nil {
	  log.Print(err)
  }
	fmt.Println(bytesThing)
  if len(bytesThing) <= 0 {
	  log.Fatal(fmt.Sprintf("Empty File.. %v", wd + "/" + fileName))
  }
  encoded := base64.StdEncoding.EncodeToString(bytesThing)
	fmt.Println("Making Request...")
  syncRequest := speech.SyncRecognizeRequest{
	  Audio: &speech.RecognitionAudio{
	    Content: string(encoded),
    },
	  Config: &speech.RecognitionConfig{
    Encoding: "FLAC",
    LanguageCode: "en-GB",
    MaxAlternatives: int64(10),
    ProfanityFilter: false,
    SampleRate: int64(16000),
   },
	}
  resp, err := speechSvc.Speech.Syncrecognize(&syncRequest).Do()
  if err != nil {
		log.Print(err)
		return nil
  }

	transcripts := []string{}
  for _, r := range resp.Results{
		for _, a := range r.Alternatives {
			transcripts = append(transcripts, a.Transcript)
    }
  }

	return transcripts
}
