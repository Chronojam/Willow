package main


import (
	"github.com/gordonklaus/portaudio"
	"strings"
	"encoding/binary"
	"encoding/base64"

	"os"
	"os/exec"
	"io/ioutil"

	"time"
	"fmt"

	context "golang.org/x/net/context"
	speech "google.golang.org/api/speech/v1beta1"
//	storage "google.golang.org/api/storage/v1"
	google "golang.org/x/oauth2/google"
)

var endpoint = "https://speech.googleapis.com/v1beta1/speech:syncrecognize"
var scope = speech.CloudPlatformScope
var bucketName = "willowbot"

func main() {
	ticker := time.NewTicker(6 * time.Second)
	quit := make(chan struct{})
	for {
		select {
			case <- ticker.C:
				var flacFile = "/tmp/sound.flac"
				fileToParse := ListenForCommand()

				fmt.Println(fmt.Sprintf("/usr/bin/avconv -i %v %v", fileToParse, flacFile))
				cmd := exec.Command("/usr/bin/avconv", "-y", "-i", fileToParse, flacFile)
				_, err := cmd.CombinedOutput()
				chk(err)
//				var fileToParse = "/home/calum/recording.flac"
				bytesThing, err := ioutil.ReadFile(flacFile)
				chk(err)
				encoded := base64.StdEncoding.EncodeToString(bytesThing)
				client, err := google.DefaultClient(context.Background(), scope)
				chk(err)

//				storageSvc, err := storage.New(client)
//				chk(err)

				speechSvc, err := speech.New(client)
				chk(err)

//				object := &storage.Object{Name: "willow-dev"}
//				file, err := os.Open(fileToParse)
//				chk(err)

//				res, err := storageSvc.Objects.Insert(bucketName, object).Media(file).Do()
//				chk(err)
//				fmt.Println(res.SelfLink)
//				var location = "gs://willowbot/willow-dev"

				syncRequest := speech.SyncRecognizeRequest{
					Audio: &speech.RecognitionAudio{
						Content: string(encoded),
					},
					Config: &speech.RecognitionConfig{
						Encoding: "FLAC",
						LanguageCode: "en-GB",
						MaxAlternatives: int64(10),
						ProfanityFilter: false,
						SampleRate: int64(44100),
					},
				}
				resp, err := speechSvc.Speech.Syncrecognize(&syncRequest).Do()
				fmt.Println(resp)
				chk(err)

				for _, r := range resp.Results{
					for _, a := range r.Alternatives {
						fmt.Println(a.Transcript)
					}
				}

		// Cleanup
//				chk(storageSvc.Objects.Delete(bucketName, "willow-dev").Do())

			case <- quit:
				ticker.Stop()
				return
		}
	}
}

func chk (err error) {
	if err != nil {
		panic(err)
	}
}


func ListenForCommand() (fn string) {
	var fileName = "/tmp/sound.aiff"
	if !strings.HasSuffix(fileName, ".aiff") {
		fileName += ".aiff"
	}

	f, err := os.Create(fileName)
	chk(err)

	_, err = f.WriteString("FORM")
	chk(err)
	chk(binary.Write(f, binary.BigEndian, int32(0)))
	_, err = f.WriteString("AIFF")
	chk(err)

	_, err = f.WriteString("COMM")
	chk(err)
	chk(binary.Write(f, binary.BigEndian, int32(18)))                  //size
  chk(binary.Write(f, binary.BigEndian, int16(1)))                   //channels
	chk(binary.Write(f, binary.BigEndian, int32(0)))                   //number of samples
	chk(binary.Write(f, binary.BigEndian, int16(32)))                  //bits per sample
	_, err = f.Write([]byte{0x40, 0x0e, 0xac, 0x44, 0, 0, 0, 0, 0, 0}) //80-bit sample rate 44100
	chk(err)

	// sound chunk
	_, err = f.WriteString("SSND")
	chk(err)
	chk(binary.Write(f, binary.BigEndian, int32(0))) //size
	chk(binary.Write(f, binary.BigEndian, int32(0))) //offset
	chk(binary.Write(f, binary.BigEndian, int32(0))) //block
	nSamples := 0
	defer func() {
		// fill in missing sizes
		totalBytes := 4 + 8 + 18 + 8 + 8 + 4*nSamples
		_, err = f.Seek(4, 0)
		chk(err)
		chk(binary.Write(f, binary.BigEndian, int32(totalBytes)))
		_, err = f.Seek(22, 0)
		chk(err)
		chk(binary.Write(f, binary.BigEndian, int32(nSamples)))
		_, err = f.Seek(42, 0)
		chk(err)
		chk(binary.Write(f, binary.BigEndian, int32(4*nSamples+8)))
		chk(f.Close())
	}()

	portaudio.Initialize()
	defer portaudio.Terminate()
	in := make([]int32, 64)

	dev, err := portaudio.Devices()
	chk(err)

	var theDeviceToUse = &portaudio.DeviceInfo{}

	for _, d := range dev {
		if strings.Contains(d.Name, "hw:3,0") {
			theDeviceToUse = d
			break
		}// Hardcoding
	}

	var numInputChannels	= 1
	var numOutputChannels = 0

	var inDev, outDev *portaudio.DeviceInfo
	if numInputChannels > 0 {
		inDev = theDeviceToUse
		chk(err)
	}
	if numOutputChannels > 0 {
		outDev, err = portaudio.DefaultOutputDevice()
		chk(err)
	}

	p := portaudio.HighLatencyParameters(inDev, outDev)
	p.Input.Channels = numInputChannels
	p.Output.Channels = numOutputChannels
	p.SampleRate = 44100
	p.FramesPerBuffer = 44100

	stream, err := portaudio.OpenStream(p, in)
	chk(err)
	defer stream.Close()

	quit := make(chan struct{})

	go func() {
		fmt.Println("Recording")
		time.Sleep(5 * time.Second)
		fmt.Println("Closing Quit")
		close(quit)
	}()

	chk(stream.Start())
	Loop:
		for {
			select {
				case <-quit:
					fmt.Println("Quitting Recording Loop")
					break Loop;
				default:
					chk(stream.Read())
					chk(binary.Write(f, binary.BigEndian, in))
					nSamples += len(in)
			}
		}
	chk(stream.Stop())
	fmt.Println("End of listen")
	return fileName
}
