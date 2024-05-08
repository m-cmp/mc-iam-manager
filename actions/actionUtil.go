package actions

import "net/http"

// 모든 응답을 CommonResponse로 한다.
type CommonResponse struct {
	ResponseData interface{} `json:"responseData"`
	Status       WebStatus   `json:"status"`
}

type WebStatus struct {
	StatusCode int    `json:"code"`
	Message    string `json:"message"`
}

func CommonResponseStatus(statusCode int, responseData interface{}) *CommonResponse {

	webStatus := WebStatus{
		StatusCode: statusCode,
		Message:    http.StatusText(statusCode),
	}
	return &CommonResponse{
		ResponseData: responseData,
		Status:       webStatus,
	}
}

func CommonResponseStatusInternalServerError(responseData interface{}) *CommonResponse {
	webStatus := WebStatus{
		StatusCode: http.StatusInternalServerError,
		Message:    http.StatusText(http.StatusInternalServerError),
	}
	return &CommonResponse{
		ResponseData: responseData,
		Status:       webStatus,
	}
}
