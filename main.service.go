package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"

	"github.com/hornbill/goApiLib"
)

//getCallServiceID takes the Call Record and returns a correct Service ID if one exists on the Instance
func getCallServiceID(swService string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) string {
	serviceID := ""
	serviceName := ""
	if importConf.ServiceMapping[swService] != nil {
		serviceName = fmt.Sprintf("%s", importConf.ServiceMapping[swService])

		if serviceName != "" {
			serviceID = getServiceID(serviceName, espXmlmc, buffer)
		}
	}
	return serviceID
}

//getServiceID takes a Service Name string and returns a correct Service ID if one exists in the cache or on the Instance
func getServiceID(serviceName string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) string {
	serviceID := ""
	if serviceName != "" {
		serviceIsInCache, ServiceIDCache := recordInCache(serviceName, "Service")
		//-- Check if we have cached the Service already
		if serviceIsInCache {
			serviceID = ServiceIDCache
		} else {
			serviceIsOnInstance, ServiceIDInstance := searchService(serviceName, espXmlmc, buffer)
			//-- If Returned set output
			if serviceIsOnInstance {
				serviceID = strconv.Itoa(ServiceIDInstance)
			}
		}
	}
	return serviceID
}

// seachService -- Function to check if passed-through service name is on the instance
func searchService(serviceName string, espXmlmc *apiLib.XmlmcInstStruct, buffer *bytes.Buffer) (bool, int) {
	boolReturn := false
	intReturn := 0
	//-- ESP Query for service
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Services")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_servicename", serviceName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	XMLServiceSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
	if xmlmcErr != nil {
		buffer.WriteString(loggerGen(4, "API Call Failed: Search Service ["+serviceName+"]: "+fmt.Sprintf("%v", xmlmcErr)))
		return boolReturn, intReturn
	}
	var xmlRespon xmlmcServiceListResponse

	err := xml.Unmarshal([]byte(XMLServiceSearch), &xmlRespon)
	if err != nil {
		buffer.WriteString(loggerGen(4, "Response Unmarshal Failed: Search Service ["+serviceName+"]: "+fmt.Sprintf("%v", err)))
		return boolReturn, intReturn
	}
	if xmlRespon.MethodResult != "ok" {
		buffer.WriteString(loggerGen(5, "MethodResult Not OK: Search Service ["+serviceName+"]: "+xmlRespon.State.ErrorRet))
		return boolReturn, intReturn
	}
	//-- Check Response
	if xmlRespon.ServiceName != "" {
		if strings.ToLower(xmlRespon.ServiceName) == strings.ToLower(serviceName) {
			intReturn = xmlRespon.ServiceID
			boolReturn = true
			//-- Add Service to Cache
			var newServiceForCache serviceListStruct
			newServiceForCache.ServiceID = intReturn
			newServiceForCache.ServiceName = serviceName
			newServiceForCache.ServiceBPMIncident = xmlRespon.BPMIncident
			newServiceForCache.ServiceBPMService = xmlRespon.BPMService
			newServiceForCache.ServiceBPMChange = xmlRespon.BPMChange
			newServiceForCache.ServiceBPMProblem = xmlRespon.BPMProblem
			newServiceForCache.ServiceBPMKnownError = xmlRespon.BPMKnownError
			serviceNamedMap := []serviceListStruct{newServiceForCache}
			mutexServices.Lock()
			services = append(services, serviceNamedMap...)
			mutexServices.Unlock()
			buffer.WriteString(loggerGen(1, "Service Cached ["+serviceName+"] ["+strconv.Itoa(xmlRespon.ServiceID)+"]"))
		}
	}

	//Return Service ID once cached - we can now use this in the calling function to get all details from cache
	return boolReturn, intReturn
}