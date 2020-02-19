package server

// Please look into the following blogpost:
// https://medium.com/@robiplus/golang-trick-export-for-test-aa16cbd7b8cd
// to see why this trick is needed.
var (
	ReadOrganizationID = readOrganizationID
	ReadClusterName    = readClusterName
	GetRouterIntParam  = getRouterIntParam
)
