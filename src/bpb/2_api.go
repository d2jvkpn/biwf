package bpb

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (nds *NDS) SetAPI() {
	nds.Engine.GET("/api", func(c *gin.Context) {
		code, data := ReqAPI()
		c.JSON(code, data)
	})

	nds.Engine.GET("/api/a", func(c *gin.Context) {
		code, data := nds.ReqQuery(c.DefaultQuery("item", ""), c.ClientIP())

		c.JSON(code, data)
	})

	nds.Engine.GET("/api/b/:project/:version/:action", func(c *gin.Context) {
		var (
			code             int
			err              error
			pv, action, item string
			job              *Job
			msg              *MSG
			data             interface{}
		)

		//
		msg, pv = new(MSG), c.Param("project")+"@"+c.Param("version")
		action, item = c.Param("action"), c.Query("item")

		if job, err = nds.MatchJob(pv); err != nil {
			msg.Title = err.Error()
			c.JSON(http.StatusNotFound, msg)
			return
		}

		//
		switch action {
		case "query":
			code, data = job.ReqQuery(item)
		case "progress":
			code, data = job.ReqProgress(item)
		case "var":
			code, data = job.ReqParameter(item)
		default:
			code, data = http.StatusBadRequest, msg
			msg.Title = "Valid API entries"
			msg.List = API_B_Get
		}

		c.JSON(code, data)
	})

	nds.Engine.POST("/api/b/:project/:version/:action", func(c *gin.Context) {
		var (
			code                      int
			err                       error
			pv, action, item, timeout string
			job                       *Job
			msg                       *MSG
			data                      interface{}
		)

		//
		msg, pv = new(MSG), c.Param("project")+"@"+c.Param("version")
		action = c.Param("action")
		item, timeout = c.Query("item"), c.DefaultQuery("timeout", "0")

		if job, err = nds.MatchJob(pv); err != nil {
			msg.Title = err.Error()
			c.JSON(http.StatusNotFound, msg)
			return
		}

		//
		switch action {
		case "interrupt":
			code, data = job.ReqInterrupt(item)
		case "submit":
			code, data = job.ReqSubmit(c, timeout)
		case "leave":
			code, data = http.StatusOK, msg
			nds.CutLeaf(pv)
			msg.Title = pv + " left"
		default:
			code, data = http.StatusBadRequest, msg
			msg.Title = "Valid API entries"
			msg.List = API_B_POST
		}

		c.JSON(code, data)
	})
}
