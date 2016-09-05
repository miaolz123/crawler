package crawler

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
)

var userAgents = []string{
	"Mozilla/5.0 (Windows; U; Windows NT 6.3; en-US) AppleWebKit/532.0 (KHTML, like Gecko) Chrome/3.0.196.2 Safari/532.0",
	"Mozilla/5.0 (Windows NT 6.2; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/32.0.1667.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/52.0.2743.116 Safari/537.36",
	"Mozilla/5.0 (Windows; U; Windows NT 6.3; en-US) AppleWebKit/532.0 (KHTML, like Gecko) Chrome/3.0.197.11 Safari/532.0",
	"Mozilla/5.0 (Windows NT 6.2; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/32.0.1667.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 6.1; WOW64; rv:29.0) Gecko/20120101 Firefox/29.0",
	"Mozilla/5.0 (Windows; U; Windows NT 6.3; en-US) AppleWebKit/532.0 (KHTML, like Gecko) Chrome/4.0.201.1 Safari/532.0",
	"Mozilla/5.0 (Windows NT 5.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/31.0.1650.63 Safari/537.36",
	"Mozilla/5.0 (Windows NT 6.3; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/43.2357.125 Safari/537.36 OPR/30.0.1835.88",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 10_0 like Mac OS X) AppleWebKit/602.1.50 (KHTML, like Gecko) Version/10.0 Mobile/14A5346a Safari/602.1",
}

func (r Rule) do(ctx *Context, q Queue) (err error) {
	ctx.Request, err = http.NewRequest(strings.ToUpper(q.Method), q.URL, bytes.NewReader([]byte{}))
	if err != nil {
		return
	}
	ctx.Request.Header.Set("User-Agent", userAgents[randIn(len(userAgents))])
	ctx.Response, err = ctx.client.Do(ctx.Request)
	return
}

// FileDownload can save a file to dist
func FileDownload(fileURL, distPath string) (filePath string, err error) {
	client := http.Client{}
	req, err := http.NewRequest("GET", fileURL, bytes.NewReader([]byte{}))
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", userAgents[randIn(len(userAgents))])
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	fileNames := strings.Split(fileURL, "/")
	fileName := fileNames[len(fileNames)-1]
	if err = os.MkdirAll(distPath, os.ModePerm); err != nil {
		return
	}
	filePath = getUniqueName(distPath + fileName)
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return
}

func getUniqueName(fileName string) string {
	f, err := os.Open(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			return fileName
		}
		return getUniqueName(fileNameAddNew(fileName))
	}
	defer f.Close()
	return getUniqueName(fileNameAddNew(fileName))
}

func fileNameAddNew(fileName string) string {
	fileNames := strings.Split(fileName, ".")
	if len(fileNames) < 2 {
		return fileName + "_new"
	}
	return strings.Join(fileNames[0:len(fileNames)-1], ".") + "_new." + fileNames[len(fileNames)-1]
}
