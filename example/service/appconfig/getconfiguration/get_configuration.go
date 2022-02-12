package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-xray-sdk-go/xray"
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

	ctx, segment := xray.BeginSegment(context.Background(), "Example-GetConfiguration")
	defer segment.Close(nil)

	configurationName := fmt.Sprintf("get-configuration-%d", time.Now().Unix())
	createConfiguration(ctx, configurationName)
	getConfiguration(ctx, configurationName)
	deleteConfiguration(ctx, configurationName)
}

func createConfiguration(ctx context.Context, configurationName string) {
	appConfigAdvance, err := appconfigadvance.NewWithApplicationName(applicationName)
	if err != nil {
		panic(err)
	}

	ok, err := appConfigAdvance.CreateConfiguration(ctx, configurationName, time.Now().Format(time.RFC3339))
	if err != nil {
		panic(err)
	}

	logger.Info("CreateConfiguration ", configurationName, " ", ok)
}

func getConfiguration(ctx context.Context, configurationName string) {
	appConfig, err := appconfig.NewWithOptions(
		appconfig.WithApplicationName(applicationName),
		appconfig.WithCacheRefreshInterval(time.Second*30),
		appconfig.WithRegionName(regionName),
	)
	if err != nil {
		panic(err)
	}

	content, err := appConfig.GetConfiguration(ctx, configurationName)
	if err != nil {
		logger.Error(err)
	} else {
		logger.Info(configurationName, ": ", content)
	}
}

func deleteConfiguration(ctx context.Context, configurationName string) {
	appConfigAdvance, err := appconfigadvance.NewWithApplicationName(applicationName)
	if err != nil {
		panic(err)
	}

	ok, err := appConfigAdvance.DeleteConfiguration(ctx, configurationName)
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
