package server

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// RouterMissingParamError missing parameter in URL
type RouterMissingParamError struct {
	paramName string
}

func (e *RouterMissingParamError) Error() string {
	return fmt.Sprintf("missing param %v", e.paramName)
}

// RouterParsingError parsing error, for example string when we expected integer
type RouterParsingError struct {
	paramName  string
	paramValue interface{}
	errString  string
}

func (e *RouterParsingError) Error() string {
	return fmt.Sprintf(
		"Error during parsing param %v with value %v. Error: %v",
		e.paramName, e.paramValue, e.errString,
	)
}

// getRouterParam retrieves parameter from URL like `/organization/{org_id}`
func getRouterParam(request *http.Request, paramName string) (string, error) {
	value, found := mux.Vars(request)[paramName]
	if !found {
		return "", &RouterMissingParamError{paramName: paramName}
	}

	return value, nil
}

// getRouterIntParam retrieves parameter from URL like `/organization/{org_id}`
// and check it for being valid integer, otherwise returns error
func getRouterIntParam(request *http.Request, paramName string) (int64, error) {
	value, err := getRouterParam(request, paramName)
	if err != nil {
		return 0, err
	}

	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, &RouterParsingError{
			paramName: paramName, paramValue: value, errString: "integer expected",
		}
	}

	return intValue, nil
}

// getRouterPositiveIntParam retrieves parameter from URL like `/organization/{org_id}`
// and check it for being valid and positive integer, otherwise returns error
func getRouterPositiveIntParam(request *http.Request, paramName string) (int64, error) {
	value, err := getRouterIntParam(request, paramName)
	if err != nil {
		return 0, err
	}

	if value <= 0 {
		return 0, &RouterParsingError{
			paramName: paramName, paramValue: value, errString: "positive integer expected",
		}
	}

	return value, nil
}

// readClusterName retrieves cluster name from request
// if it's not possible, it writes http error to the writer and returns error
func readClusterName(writer http.ResponseWriter, request *http.Request) (types.ClusterName, error) {
	clusterName, err := getRouterParam(request, "cluster")
	if err != nil {
		message := fmt.Sprintf("Cluster name is not provided %v", err.Error())
		log.Println(message)
		// query parameter 'cluster' can't be found in request, which might be caused by issue in Gorilla mux
		// (not on client side)
		responses.SendInternalServerError(writer, message)

		return "", err
	}

	if _, err := uuid.Parse(clusterName); err != nil {
		const message = "cluster name format is invalid"

		log.Println(message)
		responses.SendInternalServerError(writer, message)

		return types.ClusterName(""), errors.New(message)
	}

	return types.ClusterName(clusterName), nil
}

// readOrganizationID retrieves organization id from request
// if it's not possible, it writes http error to the writer and returns error
func readOrganizationID(writer http.ResponseWriter, request *http.Request) (types.OrgID, error) {
	organizationID, err := getRouterPositiveIntParam(request, "organization")
	if err != nil {
		message := fmt.Sprintf("Error getting organization ID from request %v", err.Error())
		log.Println(message)

		if _, ok := err.(*RouterParsingError); ok {
			responses.Send(http.StatusBadRequest, writer, err.Error())
		} else {
			responses.Send(http.StatusInternalServerError, writer, err.Error())
		}

		return 0, err
	}

	return types.OrgID(int(organizationID)), nil
}
