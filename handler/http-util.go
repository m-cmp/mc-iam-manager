package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type CommonRequest struct {
	PathParams  map[string]string `json:"pathParams"`
	QueryParams map[string]string `json:"queryParams"`
	Request     interface{}       `json:"request"`
}

type WebStatus struct {
	StatusCode int    `json:"code"`
	Message    string `json:"message"`
}

func CommonHttpCaller(callMethod string, targetFwUrl string, endPoint string, commonRequest *CommonRequest, auth string) ([]byte, error) {
	pathParamsUrl := mappingUrlPathParams(endPoint, commonRequest)
	queryParamsUrl := mappingQueryParams(pathParamsUrl, commonRequest)
	requestUrl := targetFwUrl + queryParamsUrl
	response, err := CommonHttp(requestUrl, commonRequest.Request, callMethod, auth)
	return response, err
}

func CommonHttpCallerWithoutToken(callMethod string, targetFwUrl string, endPoint string, commonRequest *CommonRequest) ([]byte, error) {
	pathParamsUrl := mappingUrlPathParams(endPoint, commonRequest)
	queryParamsUrl := mappingQueryParams(pathParamsUrl, commonRequest)
	requestUrl := targetFwUrl + queryParamsUrl
	response, err := CommonHttp(requestUrl, commonRequest.Request, callMethod, "")
	return response, err
}

func mappingUrlPathParams(endPoint string, commonRequest *CommonRequest) string {
	u := endPoint
	for k, r := range commonRequest.PathParams {
		u = strings.Replace(u, "{"+k+"}", r, -1)
	}
	return u
}

func mappingQueryParams(targeturl string, commonRequest *CommonRequest) string {
	u, err := url.Parse(targeturl)
	if err != nil {
		return ""
	}
	q := u.Query()
	for k, v := range commonRequest.QueryParams {
		q.Set(string(k), v)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func CommonHttp(url string, s interface{}, httpMethod string, auth string) ([]byte, error) {
	log.Println("CommonHttp - METHOD:" + httpMethod + " => url:" + url)
	log.Println("isauth:", auth)

	jsonData, err := json.Marshal(s)
	if err != nil {
		log.Println("commonPostERR : json.Marshal : ", err.Error())
		return nil, err
	}

	req, err := http.NewRequest(httpMethod, url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("Error CommonHttp creating request:", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Add("Authorization", auth)
	}

	requestDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		log.Println("Error CommonHttp creating httputil.DumpRequest:", err)
	}
	log.Println("\n", string(requestDump))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error CommonHttp request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, ioerr := io.ReadAll(resp.Body)
	if ioerr != nil {
		log.Println("Error CommonHttp reading response:", ioerr)
	}

	if resp.StatusCode != 200 {
		return respBody, fmt.Errorf(resp.Status)
	}

	return respBody, nil
}

func isJSONResponse(body []byte) bool {
	var js map[string]interface{}
	return json.Unmarshal(body, &js) == nil
}
