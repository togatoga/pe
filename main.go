package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	baseURL = "https://dictionary.cambridge.org/pronunciation/english/"
)

func request(word string) (*goquery.Document, error) {
	url := baseURL + word
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

func scrape(doc *goquery.Document) []string {
	urlList := []string{}
	doc.Find(".pronunciation-item").Each(func(i int, s *goquery.Selection) {
		url, _ := s.Find("span").Attr("data-src-mp3")
		if url != "" {
			urlList = append(urlList, url)
		}
	})
	return urlList
}

func download(word, url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	tmpDir := os.TempDir()
	fileName := filepath.Join(tmpDir, word+".mp3")
	file, err := os.Create(fileName)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", nil
	}
	return fileName, nil
}

func play(file string) error {
	var playCmd string
	if _, err := exec.LookPath("mpg123"); err == nil {
		playCmd = "mpg123"
	} else if _, err := exec.LookPath("afplay"); err == nil {
		playCmd = "afplay"
	}
	//play command is not found
	if playCmd == "" {
		return errors.New("Play command(mpg123, afplay) is not found")
	}
	err := exec.Command(playCmd, file).Run()
	if err != nil {
		return err
	}
	return nil
}

func run(args []string) int {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Empty word\n")
		fmt.Fprintf(os.Stderr, "pe [word]\n")
		return 1
	}

	word := strings.ToLower(args[1])
	fmt.Printf("Searching for %s\n", word)
	doc, err := request(word)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		return 1
	}

	urlList := scrape(doc)
	if len(urlList) == 0 {
		fmt.Printf("%s is not found in dictionary\n", word)
		return 0
	}

	file, err := download(word, urlList[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		return 1
	}

	fmt.Println("Playing...  Press Ctrl-C to stop.")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT)
	for {
		select {
		case <-sig:
			return 0
		default:
			time.Sleep(time.Second * 1)
			err = play(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, err.Error())
				return 1
			}

		}
	}
}

func main() {
	os.Exit(run(os.Args))
}
