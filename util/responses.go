package util

import "github.com/gin-gonic/gin"

const (
	STATUS_SUCCESS string = "SUCCESS"
	STATUS_FAILED  string = "FAILED"
)

func SuccessResponse(user any) gin.H {
	switch v := user.(type) {
	case string:
		return gin.H{
			"data":   v,
			"status": STATUS_SUCCESS,
		}
	case []interface{}:
		return gin.H{
			"data":   v,
			"status": STATUS_SUCCESS,
		}

	case map[string]interface{}:
		return gin.H{
			"data":   v,
			"status": STATUS_SUCCESS,
		}
	case []string:
		return gin.H{
			"data":   v,
			"status": STATUS_SUCCESS,
		}
	}
	return gin.H{
		"data":   "invalid data",
		"status": STATUS_FAILED,
	}
}
func FailedResponse(err error) gin.H {
	return gin.H{
		"error":  err.Error(),
		"status": STATUS_FAILED,
	}
}
