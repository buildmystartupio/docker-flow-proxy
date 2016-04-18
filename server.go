package main

import (
	"net/http"
	"fmt"
	"encoding/json"
	"strings"
	"strconv"
)

type Serverable interface {
	Execute(args []string) error
	ServeHTTP(w http.ResponseWriter, req *http.Request)
}

type Server struct {
	IP		string	`short:"i" long:"ip" default:"0.0.0.0" env:"IP" description:"IP the server listens to."`
	Port	string  `short:"p" long:"port" default:"8080" env:"PORT" description:"Port the server listens to."`
	BaseReconfigure
}


var server = Server{}

type Response struct {
	Status    		string
	Message 		string
	ServiceName		string
	ServiceColor  	string
	ServicePath   	[]string
	ServiceDomain 	string
	PathType      	string
	SkipCheck		bool
}

func (m Server) Execute(args []string) error {
	logPrintf("Starting HAProxy")
	NewRun().Execute([]string{})
	address := fmt.Sprintf("%s:%s", m.IP, m.Port)
	logPrintf(`Starting "Docker Flow: Proxy"`)
	return httpListenAndServe(address, m)
}

func (m Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	logPrintf("Processing request %s", req.URL)
	switch req.URL.Path {
	case "/v1/docker-flow-proxy/reconfigure":
		sr := ServiceReconfigure{
			ServiceName: req.URL.Query().Get("serviceName"),
			ServiceColor: req.URL.Query().Get("serviceColor"),
			ServicePath: strings.Split(req.URL.Query().Get("servicePath"), ","),
			ServiceDomain: req.URL.Query().Get("serviceDomain"),
			PathType: req.URL.Query().Get("pathType"),
		}
		if len(req.URL.Query().Get("skipCheck")) > 0 {
			sr.SkipCheck, _ = strconv.ParseBool(req.URL.Query().Get("skipCheck"))
		}
		response := Response{
			Status: "OK",
			ServiceName: sr.ServiceName,
			ServiceColor: sr.ServiceColor,
			ServicePath: sr.ServicePath,
			ServiceDomain: sr.ServiceDomain,
			PathType: sr.PathType,
			SkipCheck: sr.SkipCheck,
		}
		if len(sr.ServiceName) == 0 || len(sr.ServicePath) == 0 {
			response.Status = "NOK"
			response.Message = "The following queries are mandatory: serviceName and servicePath"
			w.WriteHeader(http.StatusBadRequest)
		} else {
			action := NewReconfigure(
				m.BaseReconfigure,
				sr,
			)
			if err := action.Execute([]string{}); err != nil {
				response.Status = "NOK"
				response.Message = fmt.Sprintf("%s", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
		httpWriterSetContentType(w, "application/json")
		js, _ := json.Marshal(response)
		w.Write(js)
	case "/v1/docker-flow-proxy/remove":
		serviceName := req.URL.Query().Get("serviceName")
		response := Response{
			Status: "OK",
			ServiceName: serviceName,
		}
		if len(serviceName) == 0 {
			response.Status = "NOK"
			response.Message = "The following queries are mandatory: serviceName and servicePath"
			w.WriteHeader(http.StatusBadRequest)
		} else {
			action := NewRemove(
				serviceName,
				m.BaseReconfigure.ConfigsPath,
				m.BaseReconfigure.TemplatesPath,
			)
			action.Execute([]string{})
		}
		httpWriterSetContentType(w, "application/json")
		js, _ := json.Marshal(response)
		w.Write(js)
	case "/v1/test", "/v2/test":
		js, _ := json.Marshal(Response{Status: "OK"})
		httpWriterSetContentType(w, "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(js)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}
