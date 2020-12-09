package main

import (
	"database/sql"
)

func getPersonalGroupByDisplayName(name string) string {
	return "groupId"
}

func checkGroupById(groupId string) bool {
	return true
}

func getResource(groupId,parentId string, path []string) string {

	return "parentId"
}

func groupIsExist(DB *sql.DB, groupId int64) *Group {
	group := getGroupByGroupId(DB, groupId)
	return group
}

func pathIsExist(groupId, parentId, resourceType, path string) bool {
	//QueryUploadPathExist(groupId, parentId, resourceType, path)
	return true
}

func findParentIdByPath(dir string) string {
	return "groupId"
}