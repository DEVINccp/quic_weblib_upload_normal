package main

import (
	"github.com/lucas-clemente/quic-go/example/crypt"
	"golang.org/x/net/html/charset"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const (
	SC_OK = 200
	SC_PARTIAL_CONTENT = 206
)

func downloadFile(response *http.ResponseWriter, request *http.Request) {
	writer := *response
	query:= request.URL.Query()
	resourceId, _ := strconv.ParseInt(query.Get("id"),10,64)
	if resourceId == 0 {
		writer.Write([]byte("Resource Id is Null! Can not download!"))
		log.Fatal("Resource Id is Null!")
	}
	database, err := ConnectWebLibDatabase()
	defer database.Close()
	if err != nil {
		log.Fatal("Connect weblib failed!\n")
		return
	}

	downloadResourceInfo := queryResourceByResourceId(database, resourceId)
	downloadPath := getDownloadResourceFullPath(downloadResourceInfo)
	exist := checkFilePathExist(downloadPath)
	if !exist {
		writer.Write([]byte("File Not Exist!"))
		log.Fatal("File Not Exist!")
	}
	byteRange := request.Header.Get("Range")
	//TO-DO
	downloadResourceInfo.suffix=strings.ToUpper(downloadResourceInfo.suffix)
	ignoreEncrypt := strings.Contains(crypt.IgnoreEncryptFile, downloadResourceInfo.suffix)
	var byteStart int64
	var byteEnd int64
	byteStart = 0
	byteEnd = downloadResourceInfo.detailSize
	if byteRange != NULL {
		split := strings.Split(byteRange, "=")
		splitByteRange := strings.Split(split[1], "-")
		byteStart, _ = strconv.ParseInt(splitByteRange[0], 10, 64)
		byteEnd, _ = strconv.ParseInt(splitByteRange[1], 10, 64)
		byteEnd = byteEnd + 1
	}
	readByteCount := byteEnd - byteStart
	writer.Header().Set("Content-Length", string(readByteCount))
	writer.Header().Add("Accept-Ranges", "bytes")
	writer.Header().Add("Last-Modified", downloadResourceInfo.lastModified)
	if readByteCount == downloadResourceInfo.detailSize {
		writer.WriteHeader(SC_OK)
	} else {
		writer.Header().Add("Content-Range","bytes " + string(byteStart) + "-" + string(byteEnd - 1))
		writer.WriteHeader(SC_PARTIAL_CONTENT)
	}
	contentType := setContentType(downloadResourceInfo.name)
	writer.Header().Set("Content-Type",contentType + "; charset=utf-8")
	isInline, _ := strconv.Atoi(query.Get("isInline"))
	toFileName := encode(downloadResourceInfo.name, request.Header.Get("User-Agent"))
	if isInline == 1 {
		writer.Header().Set("Content-Disposition", "inline; " + toFileName)
	} else {
		writer.Header().Set("Content-Disposition", "attachment; " + toFileName)
	}
	if !ignoreEncrypt {
		byteStart = byteStart + 48
	}

	file, _ := os.OpenFile(downloadPath, os.O_RDONLY, 0777)
	defer file.Close()
	_, _ = file.Seek(byteStart, 0)
	buffer := make([]byte, 1024 * 1024)
	var tempPointer int64
	if !ignoreEncrypt {
		tempPointer = byteStart - 48
	}else {
		tempPointer = byteStart
	}

	for {
		read, err := file.Read(buffer)

		if read != 0 {
			if !ignoreEncrypt {
				crypt.FileDecrypt(&buffer,tempPointer)
			}
			if readByteCount < int64(read) {
				writer.Write(buffer[:read])
				break
			} else {
				writer.Write(buffer[:read])
				readByteCount = readByteCount - int64(read)
			}
		}
		if read <= 0 {
			break
		}
		if err == io.EOF {
			break
		}
	}

}

func getDownloadResourceFullPath(resource *Resource) string {
	return UPLOAD_PATH_ROOT + "/" + strconv.FormatInt(resource.memberId, 10) + resource.filePath
}

func setContentType(fileName string) string {
	var contentType string
	contentType = "application/octet-stream"
	index := strings.LastIndex(fileName, ".")
	if index < 0 {
		return contentType
	}
	fileName = strings.ToLower(fileName)
	splitFileName := strings.Split(fileName, ".")
	if strings.EqualFold(splitFileName[1],"html") || strings.EqualFold(splitFileName[1],"htm") || strings.EqualFold(splitFileName[1],"shtml") {
		contentType = "text/html"
	} else if strings.EqualFold(splitFileName[1],"css") {
		contentType = "text/css"
	} else if strings.EqualFold(splitFileName[1],"xml") {
		contentType = "text/xml"
	}else if strings.EqualFold(splitFileName[1],"gif") {
		contentType = "image/gif"
	}else if strings.EqualFold(splitFileName[1],"jpeg") || strings.EqualFold(splitFileName[1],"jpg") {
		contentType = "image/jpeg"
	}else if strings.EqualFold(splitFileName[1],"js") {
		contentType = "application/x-javascript"
	}else if strings.EqualFold(splitFileName[1],"atom") {
		contentType = "application/atom+xml"
	}else if strings.EqualFold(splitFileName[1],"rss") {
		contentType = "application/rss+xml"
	} else if strings.EqualFold(splitFileName[1],"mml") {
		contentType = "text/mathml"
	} else if strings.EqualFold(splitFileName[1],"txt"){
		contentType = "text/plain"
	} else if strings.EqualFold(splitFileName[1],"jad"){
		contentType = "text/vnd.sun.j2me.app-descriptor"
	} else if strings.EqualFold(splitFileName[1],"wml"){
		contentType = "text/vnd.wap.wml"
	} else if strings.EqualFold(splitFileName[1],"htc"){
		contentType = "text/x-component"
	} else if strings.EqualFold(splitFileName[1],"png"){
		contentType = "image/png"
	} else if strings.EqualFold(splitFileName[1],"tif") || strings.EqualFold(splitFileName[1],"tiff") {
		contentType = "image/tiff"
	} else if strings.EqualFold(splitFileName[1],"wbmp") {
		contentType = "image/vnd.wap.wbmp"
	} else if strings.EqualFold(splitFileName[1],"ico") {
		contentType = "image/x-icon"
	} else if strings.EqualFold(splitFileName[1],"jng"){
		contentType = "image/x-jng"
	} else if strings.EqualFold(splitFileName[1],"bmp"){
		contentType = "image/x-ms-bmp"
	} else if strings.EqualFold(splitFileName[1],"svg"){
		contentType = "image/svg+xml"
	} else if strings.EqualFold(splitFileName[1],"jar") || strings.EqualFold(splitFileName[1],"var") || strings.EqualFold(splitFileName[1],"ear"){
		contentType = "application/java-archive"
	} else if strings.EqualFold(splitFileName[1],"doc"){
		contentType = "application/msword"
	} else if strings.EqualFold(splitFileName[1],"pdf"){
		contentType = "application/pdf"
	} else if strings.EqualFold(splitFileName[1],"rtf"){
		contentType = "application/rtf"
	} else if strings.EqualFold(splitFileName[1],"xls"){
		contentType = "application/vnd.ms-excel";
	} else if strings.EqualFold(splitFileName[1],"ppt"){
		contentType = "application/vnd.ms-powerpoint"
	} else if strings.EqualFold(splitFileName[1],"7z"){
		contentType = "application/x-7z-compressed"
	} else if strings.EqualFold(splitFileName[1],"rar"){
		contentType = "application/x-rar-compressed"
	} else if strings.EqualFold(splitFileName[1],"swf"){
		contentType = "application/x-shockwave-flash"
	} else if strings.EqualFold(splitFileName[1],"rpm"){
		contentType = "application/x-redhat-package-manager"
	} else if strings.EqualFold(splitFileName[1],"der") || strings.EqualFold(splitFileName[1],"pem") || strings.EqualFold(splitFileName[1],"crt") {
		contentType = "application/x-x509-ca-cert"
	} else if strings.EqualFold(splitFileName[1],"xhtml") {
		contentType = "application/xhtml+xml"
	} else if strings.EqualFold(splitFileName[1],"zip") {
		contentType = "application/zip"
	} else if strings.EqualFold(splitFileName[1],"mid") || strings.EqualFold(splitFileName[1],"midi") || strings.EqualFold(splitFileName[1],"kar") {
		contentType = "audio/midi"
	} else if strings.EqualFold(splitFileName[1],"mp3") {
		contentType = "audio/mpeg"
	} else if strings.EqualFold(splitFileName[1],"ogg") {
		contentType = "audio/ogg"
	} else if strings.EqualFold(splitFileName[1],"m4a") {
		contentType = "audio/x-m4a"
	} else if strings.EqualFold(splitFileName[1],"ra") {
		contentType = "audio/x-realaudio"
	} else if strings.EqualFold(splitFileName[1],"3gpp") || strings.EqualFold(splitFileName[1],"3gp") {
		contentType = "video/3gpp"
	} else if strings.EqualFold(splitFileName[1],"mp4") {
		contentType = "video/mp4"
	} else if strings.EqualFold(splitFileName[1],"mpeg") || strings.EqualFold(splitFileName[1],"mpg") {
		contentType = "video/mpeg"
	} else if strings.EqualFold(splitFileName[1],"mov") {
		contentType = "video/quicktime"
	} else if strings.EqualFold(splitFileName[1],"flv") {
		contentType = "video/x-flv"
	} else if strings.EqualFold(splitFileName[1],"m4v") {
		contentType = "video/x-m4v"
	} else if strings.EqualFold(splitFileName[1],"mng") {
		contentType = "video/x-mng"
	} else if strings.EqualFold(splitFileName[1],"asx") || strings.EqualFold(splitFileName[1],"asf") {
		contentType = "video/x-ms-asf"
	} else if strings.EqualFold(splitFileName[1],"wmv") {
		contentType = "video/x-ms-wmv"
	} else if strings.EqualFold(splitFileName[1],"avi") {
		contentType = "video/x-msvideo"
	}
	return contentType
}

func encode(filename, userAgent string) string {
	rtn := "filename=\"" + filename + "\""
	if userAgent != NULL {
		userAgent := strings.ToLower(userAgent)
		if strings.Contains(userAgent,"msie") {
			rtn = "filename=\"" + filename + "\""
		} else if strings.Contains(userAgent, "opera") {
			rtn = "filename*=UTF-8''" + filename
		}else if strings.Contains(userAgent, "safari") {
			rtn = "filename*=UTF-8''" + filename
		}else if strings.Contains(userAgent, "mozilla") {
			rtn = "filename*=UTF-8''" + filename
		}else if strings.Contains(userAgent, "applewebkit") {
			reader, _ := charset.NewReader(strings.NewReader("iso88591"), filename)
			all, _ := ioutil.ReadAll(reader)
			rtn = "filename=\"" + string(all) + "\""
		}
	}
	return rtn
}
