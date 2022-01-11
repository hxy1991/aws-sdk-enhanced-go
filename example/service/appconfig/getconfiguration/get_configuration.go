package main

import (
	"fmt"
	"os"
	"time"

	appconfigadvance "github.com/hxy1991/aws-sdk-enhanced-go/service/appconfig/advance"

	"github.com/hxy1991/aws-sdk-enhanced-go/awsenhanced/constant"
	"github.com/hxy1991/aws-sdk-enhanced-go/awsenhanced/logger"
	"github.com/hxy1991/aws-sdk-enhanced-go/service/appconfig"
)

var regionName = "us-east-1"
var applicationName = "app1"
var environmentName = "Test"

func main() {
	if setEnv(constant.RegionEnvName, regionName) {
		return
	}

	if setEnv(constant.EnvironmentEnvName, environmentName) {
		return
	}

	configurationName := fmt.Sprintf("get-configuration-%d", time.Now().Unix())
	createConfiguration(configurationName)
	getConfiguration(configurationName)
	deleteConfiguration(configurationName)
}

func createConfiguration(configurationName string) {
	appConfigAdvance, err := appconfigadvance.NewWithApplicationName(applicationName)
	if err != nil {
		panic(err)
	}

	ok, err := appConfigAdvance.CreateConfiguration(configurationName, time.Now().Format(time.RFC3339))
	if err != nil {
		panic(err)
	}

	logger.Info("CreateConfiguration ", configurationName, " ", ok)
}

func getConfiguration(configurationName string) {
	appConfig, err := appconfig.NewWithOptions(
		appconfig.WithApplicationName(applicationName),
		appconfig.WithCacheRefreshInterval(time.Second*30),
		appconfig.WithRegionName(regionName),
	)
	if err != nil {
		panic(err)
	}

	content, err := appConfig.GetConfiguration(configurationName)
	if err != nil {
		logger.Error(err)
	} else {
		logger.Info(configurationName, ": ", content)
	}
}

func deleteConfiguration(configurationName string) {
	appConfigAdvance, err := appconfigadvance.NewWithApplicationName(applicationName)
	if err != nil {
		panic(err)
	}

	ok, err := appConfigAdvance.DeleteConfiguration(configurationName)
	if err != nil {
		panic(err)
	}

	logger.Info("DeleteConfiguration ", configurationName, " ", ok)
}

func setEnv(key, value string) bool {
	err := os.Setenv(key, value)
	if err != nil {
		logger.Error(err)
		return true
	}
	return false
}
