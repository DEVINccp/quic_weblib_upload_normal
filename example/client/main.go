package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"flag"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/lucas-clemente/quic-go/internal/testdata"
	"github.com/lucas-clemente/quic-go/internal/utils"
	"github.com/lucas-clemente/quic-go/logging"
	"github.com/lucas-clemente/quic-go/qlog"
)

const (
	CHECKFILESTATEURL = "https://192.168.1.199:6121/demo/checkFileState"
	CHECKINDEX = "https://192.168.1.199:6121/demo/checkIndex"
	INDEXUPLOAD = "https://192.168.1.199:6121/demo/indexUpload"
	FILESTATERECORD = "https://192.168.1.199:6121/demo/fileStateRecord"
	FILEUPLOADSUCCESS = "2"
	FILEUPLOADSECTION = "1"
	FILEUPLOADNONE = "0"
	SUCCESS = "success"
	FAIL = "fail"
	FRAGMENTSIZE = 5242880
)

func main() {
	verbose := flag.Bool("v", false, "verbose")
	quiet := flag.Bool("q", false, "don't print the data")
	keyLogFile := flag.String("keylog", "", "key log file")
	insecure := flag.Bool("insecure", true, "skip certificate verification")
	enableQlog := flag.Bool("qlog", false, "output a qlog (in the same directory)")
	//filePath := flag.String("filePath","","upload file path")
	//testResultStorePath := flag.String("testResult","","test result store path")
	flag.Parse()
	urls := flag.Args()
	filePath1 := "/home/chengpingcai/Pictures/1M.zip"
	filePath := &filePath1
	testResultStorePath1 := "/home/chengpingcai/Pictures/testResult.txt"
	testResultStorePath := &testResultStorePath1
	logger := utils.DefaultLogger

	if *verbose {
		logger.SetLogLevel(utils.LogLevelDebug)
	} else {
		logger.SetLogLevel(utils.LogLevelInfo)
	}
	logger.SetLogTimeFormat("")

	var keyLog io.Writer
	if len(*keyLogFile) > 0 {
		f, err := os.Create(*keyLogFile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		keyLog = f
	}

	pool, err := x509.SystemCertPool()
	if err != nil {
		log.Fatal(err)
	}
	testdata.AddRootCA(pool)

	var qconf quic.Config
	if *enableQlog {
		qconf.Tracer = qlog.NewTracer(func(_ logging.Perspective, connID []byte) io.WriteCloser {
			filename := fmt.Sprintf("client_%x.qlog", connID)
			f, err := os.Create(filename)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Creating qlog file %s.\n", filename)
			return utils.NewBufferedWriteCloser(bufio.NewWriter(f), f)
		})
	}
	roundTripper := &http3.RoundTripper{
		TLSClientConfig: &tls.Config{
			RootCAs:            pool,
			InsecureSkipVerify: *insecure,
			KeyLogWriter:       keyLog,
		},
		QuicConfig: &qconf,
	}
	defer roundTripper.Close()
	hClient := &http.Client{
		Transport: roundTripper,
	}

	var wg sync.WaitGroup
	wg.Add(len(urls))

	fileMd5 := getFileMd5(*filePath)
	start := time.Now()
	fileState, uuid := getFileState(fileMd5, hClient)
	fmt.Println("uuid:getFileState:"+uuid)
	contentType, err := getFileContentType(*filePath)
	fmt.Println(contentType)
	count := getFileFragmentCount(*filePath)
	switch fileState {
	case FILEUPLOADNONE:
		info := writeFileInfo(fileMd5, *filePath, uuid, hClient)
		if !info {
			fmt.Println("Upload file failed")
			break
		}
		for i := 1; i<=count; i++ {
			uploadIndex(*filePath, uuid, i, hClient)
		}

	case FILEUPLOADSECTION:
		for i := 1; i <= count; i++ {
			result := checkIndex(i, uuid, hClient)
			if !result {
				for j := i; j<=count; j++ {
					fmt.Println("uuid:"+uuid)
					uploadIndex(*filePath, uuid, j, hClient)
				}
				break
			}
		}
	case FILEUPLOADSUCCESS:
		fmt.Println("Upload file success")
	}

	end := time.Since(start)
	fmt.Println("Upload time:",end)
	result := "filePath:"+(*filePath)+"; fileMds:"+fileMd5+"; fileContentType:"+contentType+"; fileLength:"+strconv.Itoa(int(getFileLength(*filePath)))+"; time:"+end.String()+"\n"
	writeTestResultIntoFile(*testResultStorePath,result)

	for _, addr := range urls {
		logger.Infof("GET %s", addr)
		go func(addr string) {
			rsp, err := hClient.Get(addr)
			if err != nil {
				log.Fatal(err)
			}
			logger.Infof("Got response for %s: %#v", addr, rsp)

			body := &bytes.Buffer{}
			_, err = io.Copy(body, rsp.Body)
			fmt.Println(body)
			if err != nil {
				log.Fatal(err)
			}
			if *quiet {
				logger.Infof("Request Body: %d bytes", body.Len())
			} else {
				logger.Infof("Request Body:")
				logger.Infof("%s", body.Bytes())
			}
			wg.Done()
		}(addr)
	}
	wg.Wait()
}

func getFileMd5(filePath string) string {
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0666)
	if err != nil {
		log.Fatal("Open file failed")
	}
	defer file.Close()

	md5Object := md5.New()
	io.Copy(md5Object, file)
	md5 := hex.EncodeToString(md5Object.Sum(nil))
	return md5
}

func getFileState(md5 string, hClient *http.Client) (string,string) {
	request, err := http.NewRequest("GET", CHECKFILESTATEURL, nil)
	request.Header.Add("md5",md5)
	checkFileState, err := hClient.Do(request)
	if err != nil {
		log.Fatal("Get response failed")
	}
	flag := checkFileState.Header.Get("flag")
	fileId := ""
	if flag != "0" {
		fileId = checkFileState.Header.Get("uuid")
	}else {
		fileId = uuid.Must(uuid.NewV4(), nil).String()
	}
	return flag,fileId
}

func writeFileInfo(md5,filePath,uuid string, hClient *http.Client) bool {
	request, err := http.NewRequest("GET", FILESTATERECORD, nil)
	request.Header.Add("md5",md5)
	stat, _ := os.Stat(filePath)
	//byte
	fileLength := stat.Size()
	request.Header.Add("fileLength",strconv.FormatInt(fileLength,10))
	fileName := stat.Name()
	request.Header.Add("fileName",fileName)
	time := stat.ModTime().Unix()
	request.Header.Add("date", string(time))
	request.Header.Add("status","000")
	request.Header.Add("suffix",getFileSuffix(filePath))
	request.Header.Add("uuid",uuid)
	contentType, err := getFileContentType(filePath)
	if err != nil {
		log.Fatal("Get file content type failed")
	}
	request.Header.Add("fileType",contentType)
	res, err := hClient.Do(request)
	record := res.Header.Get("record")
	if record == SUCCESS {
		return true
	}
	return false
}

func getFileContentType(filePath string) (string, error) {
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0666)
	if err != nil {
		log.Fatal("Open file failed")
	}
	defer file.Close()
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return "", err
	}
	contentType := http.DetectContentType(buffer)
	return contentType, nil
}

func uploadIndex(filePath, uuid string, index int, hClient *http.Client) {
	fmt.Println("uuid:uploadIndex:"+uuid)
	buffer := make([]byte, FRAGMENTSIZE)
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0666)
	if err != nil {
		log.Fatal(err)
		return
	}
	offset := (index - 1) * FRAGMENTSIZE
	whence := 0
	_, err = file.Seek(int64(offset), whence)
	if err != nil {
		log.Fatal(err)
		return
	}

	read, err := file.Read(buffer)
	if err != nil {
		log.Fatal(err)
		return
	}
	request, err := http.NewRequest("POST", INDEXUPLOAD, bytes.NewBuffer(buffer[:read]))
	if err != nil {
		log.Fatal(err)
		return
	}

	contentType, err := getFileContentType(filePath)
	if err != nil {
		log.Fatal(err)
		return
	}
	stat, err := os.Stat(filePath)
	if err != nil {
		log.Fatal(err)
		return
	}
	fileLength := stat.Size()
	request.Header.Set("content-type", contentType)
	request.Header.Add("uuid",uuid)
	request.Header.Add("index", strconv.Itoa(index))
	request.Header.Add("indexLength", strconv.Itoa(read))
	request.Header.Add("fileLength",strconv.Itoa(int(fileLength)))
	rsp, err := hClient.Do(request)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer rsp.Body.Close()
	fmt.Println(rsp.Body)
}

func checkIndex(index int, uuid string, hClient *http.Client) bool {
	request, _ := http.NewRequest("GET", CHECKINDEX, nil)
	request.Header.Add("index", strconv.Itoa(index))
	request.Header.Add("uuid",uuid)
	rsp, _ := hClient.Do(request)
	indexState := rsp.Header.Get("isUpload")
	if indexState == SUCCESS {
		return true
	}
	if indexState == FAIL {
		return false
	}
	return false
}

func getFileFragmentCount(filePath string) int {
	stat, _ := os.Stat(filePath)
	size := stat.Size()
	count := size/FRAGMENTSIZE + 1
	return int(count)
}

func getFileSuffix(filePath string) string {
	stat, err := os.Stat(filePath)
	if err != nil {
		log.Fatal(err)
	}
	fileName := stat.Name()
	split := strings.Split(fileName, ".")
	if len(split) <= 0 {
		return ""
	}
	suffix := split[len(split)-1]
	return suffix
}

func uploadIndex1(filePath, uuid string, index int, hClient *http.Client) error{
	buffer := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(buffer)
	file, err := bodyWriter.CreateFormFile("uploadFile", filePath)
	bodyWriter.WriteField("uuid",uuid)
	bodyWriter.WriteField("index",strconv.Itoa(index))
	if err != nil {
		log.Fatal(err)
		return err
	}
	open, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer open.Close()
	_, err = io.Copy(file, open)
	if err != nil {
		log.Fatal(err)
	}
	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()
	post, err := hClient.Post(INDEXUPLOAD, contentType, buffer)
	defer post.Body.Close()
	all, err := ioutil.ReadAll(post.Body)
	fmt.Println(post.Status)
	fmt.Println(string(all))
	return nil
}

func writeTestResultIntoFile(testResultPath,result string) {
	exist := checkFilePathExist(testResultPath)
	if exist {
		file, err := os.OpenFile(testResultPath, os.O_WRONLY|os.O_APPEND, 0666)
		defer file.Close()
		if err != nil {
			log.Fatal("open file failed")
		}
		file.Write([]byte(result))
	} else {
		lastIndex := strings.LastIndex(testResultPath, "/")
		path := testResultPath[:lastIndex]
		os.MkdirAll(path,0777)
		create, err := os.Create(testResultPath)
		if err != nil {
			log.Fatal("create file error")
			return
		}
		defer create.Close()
		create.Write([]byte(result))
	}
}

func checkFilePathExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err){
			return true
		}
		return false
	}
	return true
}

func execPath() (string, error) {
	file, err := exec.LookPath(os.Args[0])
	if err != nil {
		return "", err
	}
	re, err := filepath.Abs(file)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("The path is "+re)
	return filepath.Abs(file)
}
func GetCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0])) //返回绝对路径  filepath.Dir(os.Args[0])去除最后一个元素的路径
	if err != nil {
		log.Fatal(err)
	}
	return strings.Replace(dir, "\\", "/", -1) //将\替换成/
}

func getFileLength(filePath string) int64{
	stat, err := os.Stat(filePath)
	if err != nil {
		log.Fatal(err)
		return 0
	}
	fileLength := stat.Size()
	return fileLength
}
