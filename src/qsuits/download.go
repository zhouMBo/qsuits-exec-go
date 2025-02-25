package qsuits

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type MavenSearchJson struct {
	//ResponseHeader struct {
	//	Status int `json:"status"`
	//	QTime int `json:"QTime"`
	//	Params struct {
	//		Q string `json:"q"`
	//		Core string `json:"core"`
	//		Indent string `json:"indent"`
	//		Spellcheck string `json:"spellcheck"`
	//		Fl string `json:"fl"`
	//		Start string `json:"start"`
	//		Sort string `json:"sort"`
	//		SpellcheckCount string `json:"spellcheck.count"`
	//		Rows string `json:"rows"`
	//		Wt string `json:"wt"`
	//		Version string `json:"version"`
	//	} `json:"params"`
	//} `json:"responseHeader"`
	Response struct {
		NumFound int `json:"numFound"`
		Start int `json:"start"`
		Docs []struct {
			ID string `json:"id"`
			G string `json:"g"`
			A string `json:"a"`
			LatestVersion string `json:"latestVersion"`
			RepositoryID string `json:"repositoryId"`
			P string `json:"p"`
			Timestamp int64 `json:"timestamp"`
			VersionCount int `json:"versionCount"`
			Text []string `json:"text"`
			Ec []string `json:"ec"`
		} `json:"docs"`
	} `json:"response"`
	//Spellcheck struct {
	//	Suggestions []interface{} `json:"suggestions"`
	//} `json:"spellcheck"`
}

func GetLatestVersion() (latestVersion string, err error) {

	client := &http.Client{
		Timeout: time.Minute,
	}
	resp, err := client.Get("https://search.maven.org/solrsearch/select?q=a:qsuits&start=0&rows=20")
	if err != nil {
		return string(""), err
	}
	body, err := ioutil.ReadAll(resp.Body)
	_ = resp.Body.Close
	_ = resp.Close
	var f MavenSearchJson
	err = json.Unmarshal(body, &f)
	if err != nil {
		return string(""), err
	}
	return f.Response.Docs[0].LatestVersion, nil
}

func httpClientDo(resultDir string, version string, req *http.Request) (qsuitsFilePath string, err error) {

	var jarFile string
	err = os.MkdirAll(filepath.Join(resultDir, ".qsuits"), os.ModePerm)
	if err != nil {
		return jarFile, err
	}
	client := &http.Client{
		Timeout: 10 * time.Minute,
	}
	resp, err := client.Do(req)
	if err != nil {
		return jarFile, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return jarFile, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		jarFile = filepath.Join(resultDir, ".qsuits", "qsuits-" + version + ".jar")
		err = ioutil.WriteFile(jarFile, body, 0755)
		if err != nil {
			return jarFile, err
		}
		return jarFile, nil
	} else {
		return jarFile, errors.New(resp.Status)
	}
}

func DownloadFromGithub(resultDir string, version string) (qsuitsFilePath string, err error) {

	req, err := http.NewRequest("GET", "https://github.com/NigelWu95/qiniu-suits-java/releases/download/v" +
		version + "/qsuits-" + version + ".jar", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.100 Safari/537.36")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")

	return httpClientDo(resultDir, version, req)
}

func DownloadFromMaven(resultDir string, version string) (qsuitsFilePath string, err error) {

	req, err := http.NewRequest("GET", "https://search.maven.org/remotecontent?filepath=com/qiniu/qsuits/" +
		version + "/qsuits-" + version + "-jar-with-dependencies.jar", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.100 Safari/537.36")

	return httpClientDo(resultDir, version, req)
}

func progress(end <-chan struct{}, startInfo string) {

	isDone := false
	go func() {
		<-end
		isDone = true
	}()
	for {
		fmt.Printf("\r%s", startInfo)
		time.Sleep(time.Second)
		for i := 0; i <= 5 ; i++  {
			if isDone {
				return
			}
			fmt.Print(".")
			time.Sleep(time.Second)
		}
	}
}

func Download(resultDir string, version string, isLatest bool) (qsuitsFilePath string, err error) {

	done := make(chan struct{})
	if isLatest {
		go progress(done, "latest qsuits version: " + version + " is downloading")
	} else {
		go progress(done, "qsuits version: " + version + " is downloading")
	}

	qsuitsFilePath, err = DownloadFromGithub(resultDir, version)
	if err != nil {
		fmt.Println("\rdownload is retrying from maven...")
		qsuitsFilePath, err = DownloadFromMaven(resultDir, version)
	}
	done <- struct{}{}
	close(done)
	if err == nil {
		fmt.Println(" -> finished.")
	} else {
		fmt.Print("\r")
	}
	return qsuitsFilePath, err
}

func Update(path string, version string, isLatest bool) (qsuitsFilePath string, err error) {

	qsuitsJarPath := filepath.Join(path, ".qsuits", "qsuits-" + version + ".jar")
	fileInfo, err := os.Stat(qsuitsJarPath)
	if err == nil && !fileInfo.IsDir() {
		// it is already latest version
		//return qsuitsJarPath, errors.New("it is already latest version")
		return qsuitsJarPath, nil
	} else {
		return Download(path, version, isLatest)
	}
}

func Exists(path string, version string) (isExists bool, err error) {

	qsuitsJarPath := filepath.Join(path, ".qsuits", "qsuits-" + version + ".jar")
	fileInfo, err := os.Stat(qsuitsJarPath)
	if err != nil {
		return false, err
	}
	if fileInfo == nil {
		return false, errors.New("no file info")
	}
	if fileInfo.IsDir() {
		return true, errors.New(path + " is directory")
	} else {
		return true, nil
	}
}
