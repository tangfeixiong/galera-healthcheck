package monit_client

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/cloudfoundry-incubator/galera-healthcheck/config"
	"github.com/cloudfoundry-incubator/galera-healthcheck/mysql_start_mode"
	"github.com/pivotal-golang/lager"
)

type MonitClient interface {
	StartService(startMode string) (bool, error)
	StopService() (bool, error)
	GetStatus() (string, error)
	GetLogger() lager.Logger
}

type monitClient struct {
	monitConfig config.MonitConfig
	logger      lager.Logger
	serviceName string
}

func New(monitConfig config.MonitConfig, logger lager.Logger, serviceName string) *monitClient {
	return &monitClient{
		monitConfig: monitConfig,
		logger:      logger,
		serviceName: serviceName,
	}
}

func (monitClient *monitClient) GetLogger() lager.Logger {
	return monitClient.logger
}

func (monitClient *monitClient) StartService(startMode string) (bool, error) {

	mySqlStartMode := mysql_start_mode.NewMysqlStartMode(monitClient.monitConfig.MysqlStateFilePath, startMode)
	_, err := mySqlStartMode.Start()
	if err != nil {
		monitClient.logger.Error("Failed to write state file", err)
		monitClient.logger.Info("mySqlStartMode info", lager.Data{
			"startMode":          startMode,
			"MysqlStateFilePath": monitClient.monitConfig.MysqlStateFilePath,
		})
		return false, err
	}

	resp, err := monitClient.runServiceCmd("monitor", "not monitored - monitor pending")
	return resp, err
}

func (monitClient *monitClient) StopService() (bool, error) {
	resp, err := monitClient.runServiceCmd("unmonitor", "running - unmonitor pending")
	return resp, err
}

func (monitClient *monitClient) statusLookup(s MonitStatus) (string, error) {

	var tagForService ServiceTag
	foundService := false
	for _, serviceTag := range s.Services {
		if serviceTag.Name == monitClient.serviceName {
			tagForService = serviceTag
			foundService = true
			break
		}
	}
	if foundService == false {
		return "", fmt.Errorf("Could not find process %s", monitClient.serviceName)
	}

	switch {
	case tagForService.Monitor == 0:
		return "stopped", nil
	case tagForService.Monitor == 2:
		return "starting", nil
	case tagForService.Status == 0:
		return "running", nil
	default:
		return "failing", nil
	}
}

func (monitClient *monitClient) GetStatus() (string, error) {

	statusResponse, err := monitClient.runStatusCmd()
	if err != nil {
		return "", err
	}

	monitStatus, err := ParseXML(statusResponse)
	if err != nil {
		return "", err
	}

	status, err := monitClient.statusLookup(monitStatus)
	if err != nil {
		return "", err
	}

	return status, nil
}

func (monitClient *monitClient) newUrl(endpoint string, queryParams ...url.Values) (*url.URL, error) {

	config := monitClient.monitConfig

	statusURL, err := url.Parse(fmt.Sprintf("http://%s:%d/%s", config.Host, config.Port, endpoint))
	if err != nil {
		monitClient.logger.Error("Failed to parse URL", err)
		monitClient.logger.Info("URL info", lager.Data{
			"URL": statusURL,
		})
		return nil, err
	}

	if len(queryParams) > 0 {
		statusURL.RawQuery = queryParams[0].Encode()
	}

	return statusURL, nil
}

func (monitClient *monitClient) runStatusCmd() (io.Reader, error) {

	statusURL, err := monitClient.newUrl("_status", url.Values{
		"format": []string{"xml"},
	})

	resp, err := monitClient.sendRequest(statusURL, "GET")
	if err != nil {
		return nil, err
	}

	return resp, err
}

func (monitClient *monitClient) runServiceCmd(command string, expectedSuccessResponse string) (bool, error) {
	serviceAction := fmt.Sprintf("action=%s", command)

	statusURL, err := monitClient.newUrl(monitClient.serviceName)

	respBody, err := monitClient.sendRequest(statusURL, "POST", serviceAction)

	if err != nil {
		return false, err
	}
	responseBytes, _ := ioutil.ReadAll(respBody)
	responseStr := string(responseBytes)

	if !strings.Contains(responseStr, expectedSuccessResponse) {
		monitFailure := fmt.Errorf("Monit failed to %s %s successfully", command, monitClient.serviceName)
		monitClient.logger.Error("Monit failure:", monitFailure)
		monitClient.logger.Info("request info", lager.Data{
			"response_body": string(responseBytes),
		})

		return false, monitFailure
	}

	return true, nil
}

func (monitClient *monitClient) sendRequest(statusURL *url.URL, reqMethod string, params ...string) (io.Reader, error) {
	config := monitClient.monitConfig
	client := &http.Client{}

	var err error
	var req *http.Request
	if len(params) > 0 {
		req, err = http.NewRequest(reqMethod, statusURL.String(), strings.NewReader(params[0])) //bytes.NewBufferString(params[0]))
	} else {
		req, err = http.NewRequest(reqMethod, statusURL.String(), nil)
	}

	if err != nil {
		monitClient.logger.Error("Failed to create http request", err)
		return nil, err
	}

	if reqMethod == "POST" || reqMethod == "PUT" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	monitClient.logger.Info("Forwarding request to monit API", lager.Data{
		"url": req.URL,
	})

	req.SetBasicAuth(config.User, config.Password)

	resp, err := client.Do(req)
	if err != nil {
		errMsg := fmt.Errorf("Error sending http request: %s", err.Error())
		monitClient.logger.Error(errMsg.Error(), err)
		monitClient.logger.Info("request info", lager.Data{
			"request": req.URL,
		})
		return nil, errMsg
	}

	if resp.StatusCode != 200 {
		responseBytes, _ := ioutil.ReadAll(resp.Body)
		non200Error := fmt.Errorf("Received %d response from monit: %s", resp.StatusCode, responseBytes)
		monitClient.logger.Error("Failed with non-200 response", non200Error)
		monitClient.logger.Info("", lager.Data{
			"status_code":   resp.StatusCode,
			"response_body": string(responseBytes),
		})
		return nil, non200Error
	}

	monitClient.logger.Info("Made successful request to monit API")
	return resp.Body, nil
}