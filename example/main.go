package main

import (
	"bufio"
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"github.com/lucas-clemente/quic-go/example/oauth"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	_ "net/http/pprof"
	"os"
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
	"github.com/lucas-clemente/quic-go/quictrace"

	_ "github.com/satori/go.uuid"
)

type binds []string

func (b binds) String() string {
	return strings.Join(b, ",")
}

func (b *binds) Set(v string) error {
	*b = strings.Split(v, ",")
	return nil
}

// Size is needed by the /demo/upload handler to determine the size of the uploaded file
type Size interface {
	Size() int64
}

// See https://en.wikipedia.org/wiki/Lehmer_random_number_generator
func generatePRData(l int) []byte {
	res := make([]byte, l)
	seed := uint64(1)
	for i := 0; i < l; i++ {
		seed = seed * 48271 % 2147483647
		res[i] = byte(seed)
	}
	return res
}

var tracer quictrace.Tracer

const (
	CHECK = "check"
	UPLOAD = "upload"
	ZERO = "0"
	ONE = "1"
	TWO = "2"
	FILEPATH = "/home/chengpingcai/Downloads/"
	FRAGMENTSIZE = 5242880
)

func init() {
	tracer = quictrace.NewTracer()
}

func exportTraces() error {
	traces := tracer.GetAllTraces()
	if len(traces) != 1 {
		return errors.New("expected exactly one trace")
	}
	for _, trace := range traces {
		f, err := os.Create("trace.qtr")
		if err != nil {
			return err
		}
		if _, err := f.Write(trace); err != nil {
			return err
		}
		f.Close()
		fmt.Println("Wrote trace to", f.Name())
	}
	return nil
}

type tracingHandler struct {
	handler http.Handler
}

var _ http.Handler = &tracingHandler{}

func (h *tracingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
	if err := exportTraces(); err != nil {
		log.Fatal(err)
	}
}

func checkFileExist(fileName string) bool {
	_, err := os.Stat(fileName)
	return err == nil || os.IsExist(err)
}

func setupHandler(www string, trace bool) http.Handler {
	mux := http.NewServeMux()

	if len(www) > 0 {
		mux.Handle("/", http.FileServer(http.Dir(www)))
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("%#v\n", r)
			const maxSize = 1 << 30 // 1 GB
			num, err := strconv.ParseInt(strings.ReplaceAll(r.RequestURI, "/", ""), 10, 64)
			if err != nil || num <= 0 || num > maxSize {
				w.WriteHeader(400)
				return
			}
			w.Write(generatePRData(int(num)))
		})
	}

	mux.HandleFunc("/uploadResource", func(writer http.ResponseWriter, request *http.Request) {
		//query := request.URL.Query()
		//groupId := query.Get("groupId")
		//parentId := query.Get("parentId")
		//name := query.Get("name")
		//path := query.Get("path")
		//documentType := query.Get("documentType")
		//desc := query.Get("desc")
		//currentMemberId := query.Get("currentMemberId")
		//currentMemberName := query.Get("currentMemberName")
		//uploadFile(groupId,parentId,name,path,documentType,desc,currentMemberId,currentMemberName,request)

	})

	mux.HandleFunc("/demo/checkFileState", func(writer http.ResponseWriter, request *http.Request) {
		md5Value := request.Header.Get("md5")
		database, err := ConnectDatabase()
		if err != nil {
			writer.Write([]byte("Check File State failed."))
			return
		}
		fmt.Println(md5Value)
		fileMd5State, err := QueryDataInFileState(database, md5Value)
		// If can not find data in database, we can judge the file not upload.

		if err != nil || fileMd5State == nil {
			fmt.Println("0000")
			writer.Header().Add("flag", "0")
			return
		}
		state := fileMd5State.fileState
		state = 1
		if state == 2 {
			writer.Header().Add("flag", TWO)
			writer.Write([]byte("Upload File Success"))
			return
		}

		if state == 1 {
			//count the slice has been upload
			writer.Header().Add("flag", ONE)
			writer.Header().Add("uuid",fileMd5State.uuid)
			return
		}
	})

	mux.HandleFunc("/demo/checkIndex", func(writer http.ResponseWriter, request *http.Request) {
		index := request.Header.Get("index")
		uuid := request.Header.Get("uuid")
		fragmentName := FILEPATH+"/"+uuid+"_"+index
		exist := checkFileExist(fragmentName)
		//writer.Header().Add("action","checkIndex")
		if exist {
			writer.Header().Add("isUpload","success")
		} else {
			writer.Header().Add("isUpload","fail")
		}
	})

	mux.HandleFunc("/demo/fileStateRecord", func(writer http.ResponseWriter, request *http.Request) {
		fileMd5 := request.Header.Get("md5")
		date := request.Header.Get("date")
		status := request.Header.Get("status")
		name := request.Header.Get("fileName")
		fileLength, _ := strconv.Atoi(request.Header.Get("fileLength"))
		fileType := request.Header.Get("fileType")
		suffix := request.Header.Get("suffix")
		id := request.Header.Get("uuid")
		fileInfo := &FileInfo{
			fileMd5, id, date, status, name,fileLength,fileType,suffix,
		}
		database, err := ConnectDatabase()
		if err != nil {
			writer.Write([]byte("Check File State Connect database failed."))
			return
		}
		fileState := &FileState{
			md5: fileMd5,
			fileState: 1,
			uuid: id,
		}
		InsertDataIntoFileState(database, fileState)
		InsertDataInFileInfo(database, fileInfo)
		writer.Header().Add("record","success")
		writer.Header().Add("uuid", id)
	})

	mux.HandleFunc("/demo/indexUpload", func(writer http.ResponseWriter, request *http.Request) {
		uuid := request.Header.Get("uuid")
		fmt.Println("uuid1:"+uuid)
		index, _ := strconv.Atoi(request.Header.Get("index"))
		fmt.Printf("index:%d\n",index)
		_, _ = strconv.Atoi(request.Header.Get("indexLength"))
		fileLength, _ := strconv.Atoi(request.Header.Get("fileLength"))
		all, err := ioutil.ReadAll(request.Body)
		if err != nil {
			if err != io.EOF {
				log.Fatal(err)
				return
			}
		}

		fmt.Println("uuid2:"+uuid)
		storePath := FILEPATH + uuid + "_" + strconv.Itoa(index)
		newIndex, err := os.Create(storePath)
		defer newIndex.Close()
		if err != nil {
			log.Fatal(err)
			return
		}
		_, err = newIndex.Write(all)

		//merge all fragment
		indexCountTotal := (fileLength-1)/FRAGMENTSIZE +1
		fmt.Printf("Index Count Total:%d\n",indexCountTotal)
		if index == indexCountTotal {
			writer.Write([]byte("Upload Success"))
			fmt.Println("uuid3:"+uuid)
			MergeFile(uuid,fileLength)
			database, _ := ConnectDatabase()
			UpdateDataInFileState(database, uuid)
		}
	})

	mux.HandleFunc("/demo/tile", func(w http.ResponseWriter, r *http.Request) {
		// Small 40x40 png
		w.Write([]byte{
			0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
			0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x28, 0x00, 0x00, 0x00, 0x28,
			0x01, 0x03, 0x00, 0x00, 0x00, 0xb6, 0x30, 0x2a, 0x2e, 0x00, 0x00, 0x00,
			0x03, 0x50, 0x4c, 0x54, 0x45, 0x5a, 0xc3, 0x5a, 0xad, 0x38, 0xaa, 0xdb,
			0x00, 0x00, 0x00, 0x0b, 0x49, 0x44, 0x41, 0x54, 0x78, 0x01, 0x63, 0x18,
			0x61, 0x00, 0x00, 0x00, 0xf0, 0x00, 0x01, 0xe2, 0xb8, 0x75, 0x22, 0x00,
			0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
		})
	})

	mux.HandleFunc("/demo/tiles", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "<html><head><style>img{width:40px;height:40px;}</style></head><body>")
		for i := 0; i < 200; i++ {
			fmt.Fprintf(w, `<img src="/demo/tile?cachebust=%d">`, i)
		}
		io.WriteString(w, "</body></html>")
	})

	mux.HandleFunc("/demo/echo", func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("error reading body while handling /echo: %s\n", err.Error())
		}
		w.Write(body)
		w.Write([]byte("Hello QUIC"))
	})

	// accept file uploads and return the MD5 of the uploaded file
	// maximum accepted file size is 1 GB
	mux.HandleFunc("/demo/upload", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.Method)
		if r.Method == http.MethodPost {
			err := r.ParseMultipartForm(1 << 30) // 1 GB
			if err == nil {
				var file multipart.File
				file, _, err = r.FormFile("uploadfile")
				if err == nil {
					var size int64
					if sizeInterface, ok := file.(Size); ok {
						size = sizeInterface.Size()
						b := make([]byte, size)
						file.Read(b)
						md5 := md5.Sum(b)
						fmt.Fprintf(w, "%x", md5)
						return
					}
					err = errors.New("couldn't get uploaded file size")
				}
			}
			if err != nil {
				utils.DefaultLogger.Infof("Error receiving upload: %#v", err)
			}
		}
		io.WriteString(w, `<html><body><form action="/demo/upload" method="post" enctype="multipart/form-data">
				<input type="file" name="uploadfile"><br>
				<input type="submit">
			</form></body></html>`)
	})

	mux.HandleFunc("/quic/info", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("info page"))
	})

	mux.HandleFunc("/uploadChunkResource", func(writer http.ResponseWriter, request *http.Request) {
		//accessToken := request.Header.Get("Authorization")
		//if accessToken == NULL {
		//	//There can redirect to login.html TO-DO
		//	writer.Write([]byte("please login!\n"))
		//	return
		//}

		//isValidAccessToken := oauth.DecodeRsaToken(accessToken)
		//if !isValidAccessToken {
		//	writer.Write([]byte("access toekn invalid!\n"))
		//	return
		//}
		fmt.Println("uploadChunkResource request!")
		contentRange := request.Header.Get("Content-Range")
		fmt.Println(contentRange)
		if contentRange != NULL {
			fmt.Println("Chunk Upload!")
			chunkUpload(request)
		} else {
			fmt.Println("Normal Upload!")
			start := time.Now()
			uploadFile(request)
			end := time.Since(start)
			filename := request.Header.Get("filename")
			size := request.Header.Get("filesize")
			testResult := "fileName:" + filename + ";fileSize:" + size + ";time:" + end.String() +";\n"
			writeTestResultIntoFile("/home/chengpingcai/Downloads/testResult.txt",testResult)
		}
		writer.Write([]byte("{\"type\":\"success\",\"code\":\"200\",\"detail\":\"ok\",\"success\":true,\"port\":6121}"))
	})

	mux.HandleFunc("/demo/download", func(writer http.ResponseWriter, request *http.Request) {
		accessToken := request.Header.Get("Authorization")
		if accessToken == NULL {
			//There can redirect to login.html TO-DO
			writer.Write([]byte("please login!\n"))
			return
		}

		isValidAccessToken := oauth.DecodeRsaToken(accessToken)
		if !isValidAccessToken {
			writer.Write([]byte("access toekn invalid!\n"))
			return
		}
		downloadFile(&writer, request)
	})

	if !trace {
		return mux
	}
	return &tracingHandler{handler: mux}
}

func main() {
	// defer profile.Start().Stop()
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	// runtime.SetBlockProfileRate(1)

	verbose := flag.Bool("v", false, "verbose")
	bs := binds{}
	flag.Var(&bs, "bind", "bind to")
	www := flag.String("www", "", "www data")
	//tcp := flag.Bool("tcp", true, "also listen on TCP")
	trace := flag.Bool("trace", false, "enable quic-trace")
	enableQlog := flag.Bool("qlog", false, "output a qlog (in the same directory)")
	flag.Parse()

	//register server to eureka
	//defaultZone := "http://192.168.1.116:7071/eureka/"
	//appName := "quic-upload"
	//port := 6121
	//renewalInterval := 10
	//durationInterval := 30
	//go func() {
	//	regist.Regist(defaultZone,appName,port,renewalInterval,durationInterval)
	//}()

	logger := utils.DefaultLogger

	if *verbose {
		logger.SetLogLevel(utils.LogLevelDebug)
	} else {
		logger.SetLogLevel(utils.LogLevelInfo)
	}
	logger.SetLogTimeFormat("")

	if len(bs) == 0 {
		bs = binds{"127.0.0.1:6121"}
	}

	handler := setupHandler(*www, *trace)
	quicConf := &quic.Config{}
	if *trace {
		quicConf.QuicTracer = tracer
	}
	if *enableQlog {
		quicConf.Tracer = qlog.NewTracer(func(_ logging.Perspective, connID []byte) io.WriteCloser {
			filename := fmt.Sprintf("server_%x.qlog", connID)
			f, err := os.Create(filename)
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("Creating qlog file %s.\n", filename)
			return utils.NewBufferedWriteCloser(bufio.NewWriter(f), f)
		})
	}

	var wg sync.WaitGroup
	wg.Add(len(bs))
	//for _, b := range bs {
	//	bCap := b
		//go func() {
			//var err error
			//if *tcp {
			//	certFile, keyFile := testdata.GetCertificatePaths()
			//	err = http3.ListenAndServe(bCap, certFile, keyFile, handler)
			//	//http3.ListenAndServeQUIC(bCap,certFile,keyFile,handler)
			//} else {
			//	server := http3.Server{
			//		Server:     &http.Server{Handler: handler, Addr: bCap},
			//		QuicConfig: quicConf,
			//	}
			//	err = server.ListenAndServeTLS(testdata.GetCertificatePaths())
			//}
	//		certFile, keyFile := testdata.GetCertificatePaths()
	//		err := http3.ListenAndServe(bCap, certFile, keyFile, handler)
	//		if err != nil {
	//			fmt.Println(err)
	//		}
	//		wg.Done()
	//	//}()
	//}

	certFile, keyFile := testdata.GetCertificatePaths()
	err := http3.ListenAndServe(bs[0], certFile, keyFile, handler)
	if err != nil {
		fmt.Println(err)
	}
	wg.Done()
	wg.Wait()
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