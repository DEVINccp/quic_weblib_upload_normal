package main

import (
	"bufio"
	"database/sql"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	NULL = ""
	UPLOAD_UNFINISH = "0"
	UPLOAD_FINISH = "1"
	RESOURCE_TYPE_FILE = 2
	RESOURCE_STATUS_NORMAL = "normal"
	PERSONAL_CATEGORY_NAME = "#person"
	ROOT_DOMAIN_NAME = "#domain"
	UPLOAD_PATH_ROOT = "/home/weblib/weblibdata"
	UPLOAD_TEMP_FOLDER = "/home/weblib/resource/weblibapps/temp"
	enableRoleModule = true
	enableDomainModule = true
)

type formFileStruct struct {
	file multipart.File
	formName string
	fileName string
	fileSize string
}

func uploadFile(request *http.Request) {
	userAgent := request.Header.Get("User-Agent")
	host := request.URL.Host
	query := request.URL.Query()
	groupId, _ := strconv.ParseInt(query.Get("groupId"), 10, 64)
	parentId, _ := strconv.ParseInt(query.Get("parentId"), 10, 64)
	lastModified := query.Get("lastModified")
	//Correspond to weblib_group table ---- display_name field
	name := query.Get("name")
	//Correspond to weblib_group table ---- name field
	memberId,_ := strconv.ParseInt(query.Get("currentMemberId"), 10, 64)
	memberName := query.Get("currentMemberName")
	//domainTags := query.Get("domainTags")

	database, err := ConnectWebLibDatabase()
	if err != nil {
		log.Fatal("Connect weblib failed!\n")
		return
	}
	var group *Group
	if groupId == 0 {
		if name == NULL {
			log.Fatal("Can not find the cabinet to store")
			return
		}
		category := getCategoryByCategoryName(database, PERSONAL_CATEGORY_NAME)
		group = getGroupByCategoryIdAndGroupDisplayName(database, category.id, name)
	} else {
		group = getGroupByGroupId(database,groupId)
	}
	if group == nil {
		log.Fatal("Can not find the user's cabinet")
		return
	}

	if isMultipart(request) {
		now := time.Now()
		reader, err := request.MultipartReader()
		if err != nil {
			panic("Get multipart Reader failed!")
		}
		form, err := reader.ReadForm(2 << 30)
		formNames := getFileNames(form.File)
		if len(*formNames) <= 0 {
			log.Fatal("resource file list is null")
			return
		}

		isPersonalGroup := false
		personGroupJudge := getGroupByMemberId(database, memberId)
		if personGroupJudge.id == group.id {
			isPersonalGroup = true
		}
		fmt.Println(isPersonalGroup)

		fileCount := getFileCountInMultiPartForm(form)
		var uploadSize int64
		for i := 0; i < fileCount; i++ {
			fileData := form.File["filedata"][i]
			filename := fileData.Filename
			fileSize := fileData.Size

			open, err := fileData.Open()
			if err != nil {
				log.Fatal("open file error!\n")
				return
			}
			data, err := ioutil.ReadAll(open)

			if err != nil {
				log.Fatal("read data from multipart file failed!\n")
				return
			}
			//Domain tag default
			//public date default
			open.Seek(0,io.SeekStart)
			contentType := getFileContentType(open)
			open.Close()
			len := fileSize / 1024
			if len * 1024 < fileSize {
				len++
			}

			var pathTmp string
			if parentId == 0 {
				pathTmp = "/"
			} else {
				//through parentId get parent directory path
				//get path
				path := getResourcePathById(database, parentId)
				pathTmp = path + strconv.FormatInt(parentId, 10) + "/"
			}
			//document type default value

			resource := &Resource{
				contentType:    contentType,
				now:            now,
				groupId: 				groupId,
				groupName:      group.name,
				memberId:       memberId,
				memberName:     memberName,
				fileOriginalName:       filename,
				parentId:       parentId,
				lastModified:   lastModified,
				resourceType:   RESOURCE_TYPE_FILE,
				resourceStatus: RESOURCE_STATUS_NORMAL,
				flag: 					UPLOAD_UNFINISH,
				size: 					len,
				detailSize: 		fileSize,
				path:           pathTmp,
				name: 					filename,
			}

			resourceInfoIntoDatabase := saveResourceInfoIntoDatabase(database, resource)
			if resourceInfoIntoDatabase == 0 {
				log.Fatal("insert into database failed!\n")
				return
			}
			resource.id = resourceInfoIntoDatabase
			perm := hasPermToUpload(database, resource)
			if !perm {
				DeleteResourceInfoByResourceId(database, resource.id)
				return
			}

			//begin upload
			beginUploadFile(data, resource, group, database, host, userAgent)
			//begin thumbnail

			uploadSize = uploadSize + resource.detailSize
			//modify parent folder used capacity
			modifyParentFolderSize(database, resource)
		}
		//modify capacity
		modifyGroupCapacityInfo(database, group, uploadSize)
	}
	defer database.Close()
}

func chunkUpload(request *http.Request) {
	contentRange := request.Header.Get("Content-Range")
	userAgent := request.Header.Get("User-Agent")
	host := request.URL.Host
	query := request.URL.Query()
	groupId, _ := strconv.ParseInt(query.Get("groupId"), 10, 64)
	parentId, _ := strconv.ParseInt(query.Get("parentId"), 10, 64)
	lastModified := query.Get("lastModified")
	//Correspond to weblib_group table ---- display_name field
	name := query.Get("name")
	//Correspond to weblib_group table ---- name field
	memberId,_ := strconv.ParseInt(query.Get("currentMemberId"), 10, 64)
	memberName := query.Get("currentMemberName")
	//domainTags := query.Get("domainTags")
	resourceId, _ := strconv.ParseInt(query.Get("resourceId"), 10, 64)

	database, err := ConnectWebLibDatabase()
	if err != nil {
		log.Fatal("Connect weblib failed!\n")
		return
	}

	var group *Group
	if groupId == 0 {
		if name == NULL {
			log.Fatal("Can not find the cabinet to store")
			return
		}
		category := getCategoryByCategoryName(database, PERSONAL_CATEGORY_NAME)
		group = getGroupByCategoryIdAndGroupDisplayName(database, category.id, name)
	} else {
		group = getGroupByGroupId(database,groupId)
	}
	if group == nil {
		log.Fatal("Can not find the user's cabinet")
		return
	}

	if isMultipart(request) {
		now := time.Now()
		reader, err := request.MultipartReader()
		if err != nil {
			panic("Get multipart Reader failed!")
		}
		form, err := reader.ReadForm(2 << 30)
		formNames := getFileNames(form.File)
		if len(*formNames) <= 0 {
			log.Fatal("resource file list is null")
			return
		}

		isPersonalGroup := false
		personGroupJudge := getGroupByMemberId(database, memberId)
		if personGroupJudge.id == group.id {
			isPersonalGroup = true
		}
		fmt.Println(isPersonalGroup)

		fileCount := getFileCountInMultiPartForm(form)
		var uploadSize int64
		for i := 0; i < fileCount; i++ {
			fileData := form.File["filedata"][i]
			filename := fileData.Filename

			open, err := fileData.Open()
			if err != nil {
				log.Fatal("open file error!\n")
				return
			}
			data, err := ioutil.ReadAll(open)

			if err != nil {
				log.Fatal("read data from multipart file failed!\n")
				return
			}

			startByte, endByte, totalSize := parseContentRange(contentRange)
			byteSize := totalSize / 1024
			if byteSize * 1024 < totalSize {
				byteSize++
			}
			if startByte == 0 {
				resource := &Resource{
					contentType:      contentRange,
					now:              now,
					groupId:          groupId,
					groupName:        group.name,
					memberId:         memberId,
					memberName:       memberName,
					fileOriginalName: filename,
					flag:             UPLOAD_UNFINISH,
					detailSize:       totalSize,
					size:             byteSize,
					parentId:         parentId,
				}
				if lastModified != NULL {
					timeLastModified, _ := time.Parse("2006-01-02 15:04:05", lastModified)
					resource.lastModified = timeLastModified.String()
				}
				var pathTmp string
				if parentId == 0 {
					pathTmp = "/"
				} else {
					//through parentId get parent directory path
					//get path
					path := getResourcePathById(database, parentId)
					pathTmp = path + strconv.FormatInt(parentId, 10) + "/"
				}
				resource.path = pathTmp
				resource.resourceType = RESOURCE_TYPE_FILE
				resource.resourceStatus = RESOURCE_STATUS_NORMAL
				resourceInfoIntoDatabase := saveResourceInfoIntoDatabase(database, resource)
				resource.id = resourceInfoIntoDatabase
				tempFolder := getChunkUploadTempFolder(resource)
				if resource == nil {
					log.Fatal("resource pointer is null!\n")
					return
				}
				resource.rate = 1
				resource.flag = ONE

				prefix, suffix := getResourceFileNamePreAndSuffix(resource.fileOriginalName)
				resource.suffix = suffix
				resource.filePreName = prefix
				fileSizeValid := checkFileSizeValid(database, resource.detailSize, group)
				if !fileSizeValid {
					DeleteResourceInfoByResourceId(database, resource.id)
				}
				groupAvailableCapacity := checkGroupAvailableCapacity(resource.detailSize, group)
				if !groupAvailableCapacity {
					DeleteResourceInfoByResourceId(database, resource.id)
				}
				renameResource(resource, suffix)
				checkResourceOriginalNameAndUpdate(database, resource)
				updateResourceByResourceId(database, resource)
				copyFileToServer(data, tempFolder, "0")
				resource.reserveField = 1
				updateResourceReserveFieldByResourceId(database, resource)
			} else if endByte + 1 == totalSize {

				resource := queryResourceByResourceId(database, resourceId)
				tempFolder := getChunkUploadTempFolder(resource)
				copyFileToServer(data, tempFolder, strconv.FormatInt(byteSize,10))
				if lastModified != NULL {
					timeLastModified, _ := time.Parse("2006-01-02 15:04:05", lastModified)
					resource.lastModified = timeLastModified.String()
				}
				resource.reserveField = resource.reserveField + 1
				updateResourceAtLastChunk(database, resource)
				mergeFile(tempFolder,resource.path,resource.filePath)
				uploadSize = uploadSize + resource.detailSize
				modifyParentFolderSize(database, resource)
				modifyGroupCapacityInfo(database, group, uploadSize)
				addUploadLog(database, resource, group, host, userAgent)
			} else {
				resource := queryResourceByResourceId(database, resourceId)
				tempFolder := getChunkUploadTempFolder(resource)
				copyFileToServer(data, tempFolder, strconv.FormatInt(byteSize,10))
				if lastModified != NULL {
					timeLastModified, _ := time.Parse("2006-01-02 15:04:05", lastModified)
					resource.lastModified = timeLastModified.String()
				}
				resource.reserveField = resource.reserveField + 1
				updateResourceAtMiddle(database,resource)
			}

		}
	}
}

//check request is post and head content multipart/form-data
func isMultipart(request *http.Request) bool{
	if strings.ToLower(request.Method) != "post" {
		return false
	} else {
		contentType := strings.ToLower(request.Header.Get("Content-Type"))
		return strings.HasPrefix(contentType, "multipart/")
	}
}

func getFileNames(formFiles map[string][]*multipart.FileHeader) *[]string{
	var fileNames []string
	for formName, _ := range formFiles {
		fileNames = append(fileNames, formName)
	}
	return &fileNames
}

func getFileCountInMultiPartForm(form *multipart.Form) int{
	allFile := form.File["filedata"]
	if len(allFile) <= 0 {
		allFile = form.File["Filedata"]
	}

	if len(allFile) <= 0 {
		log.Fatal("resource file list is null")
		return 0
	}

	return len(allFile)
}

func getFileContentType(file multipart.File) string{
	buffer := make([]byte, 1024)
	reader := bufio.NewReaderSize(file,1024)
	reader.Read(buffer)
	contentType := http.DetectContentType(buffer)
	return contentType
}

//judge file type
func copyFileToServer(data []byte, savePath, saveName string) bool{
	if !dirIsExist(savePath) {
		os.MkdirAll(savePath,0777)
	}
	if dirIsExist(savePath) {
		err := os.Chdir(savePath)
		if err != nil {
			log.Fatal("change directory error!\n")
			return false
		}
		create, err := os.Create(saveName)
		defer create.Close()
		//TO-DO CRYPT FILE
		create.Write(data)
		return true
	} else {
		log.Fatal("Path not exist")
		return false
	}
}

func dirIsExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		if os.IsNotExist(err) {
			return false
		}
		return false
	}
	return true
}

func mergeFile(tempPath, normalPath,filePath string) {
	tempPathIsExist := dirIsExist(tempPath)
	if !tempPathIsExist {
		log.Fatal("temp Path is not exist!\n")
		return
	}
	normalPathIsExist := dirIsExist(normalPath)
	if !normalPathIsExist {
		log.Fatal("normal Path is not exist!\n")
		return
	}
	dir, _ := ioutil.ReadDir(tempPath)
	var count int
	for _, _ = range dir {
		count++
	}
	normalFile, err := os.Create(normalPath + filePath)
	defer normalFile.Close()
	if err != nil {
		panic("create file failed!")
	}
	bytes := make([]byte, 1024)
	for i := 0; i < count; i++ {
		indexName := strconv.Itoa(i)
		openFile, err := os.OpenFile(tempPath+indexName, os.O_RDONLY, 0666)
		if err != nil {
			if err != io.EOF {
				log.Fatal("Open file failed")
			}
		}
		read, err := openFile.Read(bytes)
		if err != nil {
			if err != io.EOF {
				log.Fatal("Read file failed")
			}
		}
		normalFile.Write(bytes[:read])
		openFile.Close()
		os.Remove(indexName)
	}
	os.Remove(tempPath)
	//move to normal path and rename
}

func hasPermToUpload(database *sql.DB, resource *Resource) bool {
	//get person category according name(#person)
	//judge group.categoryId == category person Id
	//according member name find member

	//according member id get application
	if enableRoleModule {
		judgeMemberHasMemberRole := queryMemberIsRoleManager(database, resource)
		if judgeMemberHasMemberRole {
			return true
		}
	}

	if enableDomainModule {
		judgeMemberHasDomainRole := queryMemberIsDomainManager(database, resource)
		if judgeMemberHasDomainRole {
			return true
		}
	}

	judgeMemberIsSystemManager := queryMemberIsSystemManager(database, resource)
	if judgeMemberIsSystemManager {
		return true
	}
	return false
}

func beginUploadFile(data []byte, resource *Resource, group *Group, database *sql.DB, host, userAgent string) {
	if resource == nil {
		log.Fatal("resource pointer is null!\n")
		return
	}
	resource.rate = 0
	resource.flag = ONE

	prefix, suffix := getResourceFileNamePreAndSuffix(resource.fileOriginalName)
	resource.suffix = suffix
	resource.filePreName = prefix
	fileSizeValid := checkFileSizeValid(database, resource.detailSize, group)
	if !fileSizeValid {
		DeleteResourceInfoByResourceId(database, resource.id)
	}
	groupAvailableCapacity := checkGroupAvailableCapacity(resource.detailSize, group)
	if !groupAvailableCapacity {
		DeleteResourceInfoByResourceId(database, resource.id)
	}
	renameResource(resource, suffix)
	checkResourceOriginalNameAndUpdate(database, resource)
	path := getFileFullPath(database, resource, group)
	copyFile := copyFileToServer(data, path, resource.filePath)
	updateResourceByResourceId(database, resource)
	if data != nil {
		if !copyFile {
			DeleteResourceInfoByResourceId(database, resource.id)
		}
		addUploadLog(database, resource, group, host, userAgent)
	}
}

func getResourceFileNamePreAndSuffix(fileName string) (string,string) {
	index := strings.LastIndex(fileName, ".")
	if index == -1 {
		return fileName, ""
	} else {
		return fileName[0:index], fileName[index+1:]
	}
}

func checkFileSizeValid(database *sql.DB, fileSize int64, group *Group) bool {
	groupType := getGroupTypeByTypeId(database, group.groupType)
	if fileSize/1024 > groupType.singleFileSize {
		log.Fatal("File too large!\n")
		return false
	}
	return true
}

func checkGroupAvailableCapacity(fileSize int64, group *Group) bool {
	if fileSize > group.availableCapacity {
		log.Fatal("File too large!\n")
		return false
	}
	return true
}

func renameResource(resource *Resource, suffix string) {
	var storeName string
	uuid := uuid.Must(uuid.NewV4(), nil).String()
	storeName = strconv.FormatInt(resource.memberId, 10) + "_" + uuid
	resource.filePath = storeName + "." + suffix
}

var count int
func checkResourceOriginalNameAndUpdate(database *sql.DB,resource *Resource) {
	result := checkHaveSameFileInGroupParent(database, resource)
	if !result {
		count = 0
		return
	}
	count++
	resource.name = resource.filePreName + "(" + strconv.Itoa(count) + ")." + resource.suffix
	checkResourceOriginalNameAndUpdate(database, resource)
}

func getFileFullPath(database *sql.DB, resource *Resource, group *Group) string {
	path := isDomainGroup(database, group)
	if path == NULL {
		path = UPLOAD_PATH_ROOT + "/" + strconv.FormatInt(resource.memberId, 10) + "/"
		os.MkdirAll(path,0777)
		return path
	} else {
		log.Fatal("bad configuration for domain resource path!\n")
	}
	return NULL
}

func isDomainGroup(database *sql.DB, group *Group) string{
	category := getCategoryByCategoryName(database, ROOT_DOMAIN_NAME)
	if category == nil {
		log.Fatal("category is nil!\n")
		return ""
	}
	id := group.categoryId
	for id > 0 {
		groupCategory := getCategoryByGroupCategoryId(database, id)
		if groupCategory.parentId == category.id {
			domainCategory := getDomainCategoryByCategoryId(database, groupCategory.parentId)
			if domainCategory == nil {
				return NULL
			} else {
				return domainCategory.relativePath
			}
		} else {
			id = groupCategory.parentId
		}
	}
	return NULL
}

func modifyGroupCapacityInfo(database *sql.DB, group *Group, uploadSize int64) {
	group.availableCapacity = group.availableCapacity - uploadSize / 1024
	group.usedCapacity = group.usedCapacity + uploadSize / 1024
	updateGroupCapacityInfo(database,group)
}

func modifyParentFolderSize(database *sql.DB, resource *Resource) {
	parentId := resource.parentId
	for parentId > 0 {
		parentFolder := queryResourceById(database, resource)
		parentFolder.size = parentFolder.size + resource.size
		updateResourceSize(database, &parentFolder)
		parentId = parentFolder.parentId
	}
}

func parseContentRange(contentRange string) (int64, int64, int64) {
	//Content-Range:bytes 1048576-2097151/10259052
	splitSpace := strings.Split(contentRange, " ")
	splitRowLine := strings.Split(splitSpace[1], "-")
	startByte, _ := strconv.ParseInt(splitRowLine[0],10,64)
	splitSlantLine := strings.Split(splitRowLine[1], "/")
	endByte, _ := strconv.ParseInt(splitSlantLine[0], 10, 64)
	totalSize, _ := strconv.ParseInt(splitSlantLine[1], 10, 64)
	return startByte, endByte, totalSize
}

func getChunkUploadTempFolder(resource *Resource) string {
	path := UPLOAD_TEMP_FOLDER + "/" + strconv.FormatInt(resource.memberId, 10) + "/" + strconv.FormatInt(resource.id, 10) +  "/"
	os.MkdirAll(path,0777)
	return path
}