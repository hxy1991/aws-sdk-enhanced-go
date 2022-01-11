package appconfigadvance

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/appconfig"
	"github.com/hxy1991/aws-sdk-enhanced-go/awsenhanced/constant"
)

// AllAtOnceNotBake deployment strategy must have been created
const deploymentStrategyName = "AllAtOnceNotBake"

type EnhancedAppConfigAdvance struct {
	regionName      string
	applicationName string
	environmentName string
	applicationId   string
	environmentId   string
	appConfigClient *appconfig.AppConfig
}

var applicationNameId = map[string]string{}
var environmentNameId = map[string]string{}

var configurationProfileNameId = sync.Map{}
var deploymentStrategyNameId = sync.Map{}

func NewWithApplicationName(applicationName string) (*EnhancedAppConfigAdvance, error) {
	return NewWithOptions(WithApplicationName(applicationName))
}

func NewWithOptions(opts ...Option) (*EnhancedAppConfigAdvance, error) {
	appConfigAdvance := &EnhancedAppConfigAdvance{
		regionName:      os.Getenv(constant.RegionEnvName),
		applicationName: "",
		environmentName: os.Getenv(constant.EnvironmentEnvName),

		applicationId:   "",
		environmentId:   "",
		appConfigClient: nil,
	}

	err := appConfigAdvance.ApplyOptions(opts...)
	if err != nil {
		return nil, err
	}

	if appConfigAdvance.applicationName == "" {
		return nil, errors.New("missing required field: ApplicationName")
	}

	if appConfigAdvance.regionName == "" {
		msg := fmt.Sprintf("missing required field: RegionName or set %s env", constant.RegionEnvName)
		return nil, errors.New(msg)
	}

	if appConfigAdvance.environmentName == "" {
		msg := fmt.Sprintf("missing required field: EnvironmentName or set %s env", constant.EnvironmentEnvName)
		return nil, errors.New(msg)
	}

	if appConfigAdvance.appConfigClient == nil {
		err = appConfigAdvance.initAppConfigClient()
		if err != nil {
			return nil, err
		}
	}

	err = appConfigAdvance.listApplications()
	if err != nil {
		return nil, err
	}

	err = appConfigAdvance.listDeploymentStrategies()
	if err != nil {
		return nil, err
	}

	err = appConfigAdvance.nameToId()
	if err != nil {
		return nil, err
	}

	err = appConfigAdvance.listConfigurationProfiles()
	if err != nil {
		return nil, err
	}

	return appConfigAdvance, err
}

func (appConfigAdvance *EnhancedAppConfigAdvance) initAppConfigClient() error {
	awsConfig := aws.Config{
		Region: aws.String(appConfigAdvance.regionName),
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Config: awsConfig,
	}))

	appConfigClient := appconfig.New(sess)

	if appConfigClient == nil {
		return errors.New("can not init aws AppConfig client")
	}

	appConfigAdvance.appConfigClient = appConfigClient
	return nil
}

func (appConfigAdvance *EnhancedAppConfigAdvance) UpdateConfiguration(configurationName string, content string) (bool, error) {
	configurationProfileId, found, err := appConfigAdvance.getConfigurationProfileId(configurationName)
	if err != nil {
		return false, err
	}
	if !found {
		msg := fmt.Sprintf("configuration [%s] do not exist in [%s] environment of [%s] application", configurationName, appConfigAdvance.environmentName, appConfigAdvance.applicationName)
		return false, errors.New(msg)
	}

	// 创建版本
	createHostedConfigurationVersionOutput, err := appConfigAdvance.createHostedConfigurationVersion(configurationProfileId, content)
	if err != nil {
		return false, err
	}

	// 发布版本
	configurationVersion := fmt.Sprintf("%d", *createHostedConfigurationVersionOutput.VersionNumber)
	startDeploymentOutput, err := appConfigAdvance.startDeployment(configurationProfileId, configurationVersion)
	if err != nil {
		return false, err
	}
	return startDeploymentOutput != nil, nil
}

func (appConfigAdvance *EnhancedAppConfigAdvance) getConfigurationProfileId(configurationName string) (string, bool, error) {
	configurationProfileId, found := configurationProfileNameId.Load(configurationName)
	if found {
		return configurationProfileId.(string), found, nil
	}
	// 再获取一次配置名称和ID的对应关系
	err := appConfigAdvance.listConfigurationProfiles()
	if err != nil {
		return "", false, err
	}

	configurationProfileId, found = configurationProfileNameId.Load(configurationName)
	if found {
		return configurationProfileId.(string), found, nil
	}

	return "", false, nil
}

func (appConfigAdvance *EnhancedAppConfigAdvance) CreateConfiguration(configurationName string, content string) (bool, error) {
	_, found, err := appConfigAdvance.getConfigurationProfileId(configurationName)
	if err != nil {
		return false, err
	}
	if found {
		// 配置已经存在
		msg := fmt.Sprintf("configuration [%s] already exist in [%s] environment of [%s] application", configurationName, appConfigAdvance.environmentName, appConfigAdvance.applicationName)
		return false, errors.New(msg)
	}

	// 创建配置 Profile
	createConfigurationProfileOutput, err := appConfigAdvance.createConfigurationProfile(configurationName)
	if err != nil {
		return false, err
	}
	configurationProfileId := *createConfigurationProfileOutput.Id

	// 创建版本
	createHostedConfigurationVersionOutput, err := appConfigAdvance.createHostedConfigurationVersion(configurationProfileId, content)
	if err != nil {
		return false, err
	}

	// 发布版本
	configurationVersion := fmt.Sprintf("%d", *createHostedConfigurationVersionOutput.VersionNumber)
	startDeploymentOutput, err := appConfigAdvance.startDeployment(configurationProfileId, configurationVersion)
	if err != nil {
		return false, err
	}

	if startDeploymentOutput != nil {
		configurationProfileNameId.Store(configurationName, configurationProfileId)
	}

	return startDeploymentOutput != nil, nil
}

func (appConfigAdvance *EnhancedAppConfigAdvance) createConfigurationProfile(configurationProfileName string) (*appconfig.CreateConfigurationProfileOutput, error) {
	input := appconfig.CreateConfigurationProfileInput{
		ApplicationId: aws.String(appConfigAdvance.applicationId),
		// 目前只这种类型
		LocationUri: aws.String("hosted"),
		Name:        aws.String(configurationProfileName),
	}
	return appConfigAdvance.appConfigClient.CreateConfigurationProfile(&input)
}

func (appConfigAdvance *EnhancedAppConfigAdvance) createHostedConfigurationVersion(configurationProfileId string, content string) (*appconfig.CreateHostedConfigurationVersionOutput, error) {
	contentType := http.DetectContentType([]byte(content))
	input := appconfig.CreateHostedConfigurationVersionInput{
		ApplicationId:          aws.String(appConfigAdvance.applicationId),
		ConfigurationProfileId: aws.String(configurationProfileId),
		Content:                []byte(content),
		// text/plain; charset=UTF-8 只取 text/plain
		ContentType: aws.String(strings.SplitN(contentType, "; ", 2)[0]),
	}
	return appConfigAdvance.appConfigClient.CreateHostedConfigurationVersion(&input)
}

func (appConfigAdvance *EnhancedAppConfigAdvance) startDeployment(configurationProfileId string, configurationVersion string) (*appconfig.StartDeploymentOutput, error) {
	deploymentStrategyId, found := deploymentStrategyNameId.Load(deploymentStrategyName)
	if !found {
		msg := fmt.Sprintf("deploymentStrategy [%s] do not exist in [%s] application", deploymentStrategyName, appConfigAdvance.applicationName)
		return nil, errors.New(msg)
	}
	input := appconfig.StartDeploymentInput{
		ApplicationId:          aws.String(appConfigAdvance.applicationId),
		EnvironmentId:          aws.String(appConfigAdvance.environmentId),
		ConfigurationProfileId: aws.String(configurationProfileId),
		ConfigurationVersion:   aws.String(configurationVersion),
		DeploymentStrategyId:   aws.String(deploymentStrategyId.(string)),
	}
	return appConfigAdvance.appConfigClient.StartDeployment(&input)
}

func (appConfigAdvance *EnhancedAppConfigAdvance) DeleteConfiguration(configurationName string) (bool, error) {
	configurationProfileId, found, err := appConfigAdvance.getConfigurationProfileId(configurationName)
	if err != nil {
		return false, err
	}
	if !found {
		msg := fmt.Sprintf("configuration [%s] do not exist in [%s] environment of [%s] application", configurationName, appConfigAdvance.environmentName, appConfigAdvance.applicationName)
		return false, errors.New(msg)
	}
	output, err := appConfigAdvance.deleteConfigurationProfile(configurationProfileId)
	if err != nil {
		return false, err
	}

	if output != nil {
		configurationProfileNameId.Delete(configurationName)
	}

	return output != nil, nil
}

func (appConfigAdvance *EnhancedAppConfigAdvance) deleteConfigurationProfile(configurationProfileId string) (*appconfig.DeleteConfigurationProfileOutput, error) {
	err := appConfigAdvance.deleteAllConfigurationVersion(configurationProfileId)
	if err != nil {
		return nil, err
	}

	input := appconfig.DeleteConfigurationProfileInput{
		ApplicationId:          aws.String(appConfigAdvance.applicationId),
		ConfigurationProfileId: aws.String(configurationProfileId),
	}
	output, err := appConfigAdvance.appConfigClient.DeleteConfigurationProfile(&input)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func (appConfigAdvance *EnhancedAppConfigAdvance) deleteAllConfigurationVersion(configurationProfileId string) error {
	var nextToken *string = nil
	for {
		listHostedConfigurationVersionsInput := appconfig.ListHostedConfigurationVersionsInput{
			ApplicationId:          aws.String(appConfigAdvance.applicationId),
			ConfigurationProfileId: aws.String(configurationProfileId),
			NextToken:              nextToken,
		}
		listHostedConfigurationVersionsOutput, err := appConfigAdvance.appConfigClient.ListHostedConfigurationVersions(&listHostedConfigurationVersionsInput)
		if err != nil {
			return err
		}
		if len(listHostedConfigurationVersionsOutput.Items) == 0 {
			break
		}

		for _, item := range listHostedConfigurationVersionsOutput.Items {
			input := appconfig.DeleteHostedConfigurationVersionInput{
				ApplicationId:          aws.String(appConfigAdvance.applicationId),
				ConfigurationProfileId: item.ConfigurationProfileId,
				VersionNumber:          item.VersionNumber,
			}
			output, err := appConfigAdvance.appConfigClient.DeleteHostedConfigurationVersion(&input)
			if err != nil {
				return err
			}
			if output == nil {
				msg := fmt.Sprintf("delete hosted configuration version failed, configurationProfileId: %s, versionNumber: %d", *item.ConfigurationProfileId, *item.VersionNumber)
				return errors.New(msg)
			}
		}

		nextToken = listHostedConfigurationVersionsOutput.NextToken
		if nextToken == nil {
			break
		}
	}
	return nil
}

func (appConfigAdvance *EnhancedAppConfigAdvance) nameToId() error {
	applicationId, found := applicationNameId[appConfigAdvance.applicationName]
	if !found {
		return fmt.Errorf("can not find application [%s]", appConfigAdvance.applicationName)
	}

	err := appConfigAdvance.listEnvironments(appConfigAdvance.appConfigClient, applicationId)
	if err != nil {
		return err
	}

	environmentId, found := environmentNameId[appConfigAdvance.environmentName]
	if !found {
		return fmt.Errorf("can not find environment %s at application %s", environmentId, appConfigAdvance.applicationName)
	}

	appConfigAdvance.applicationId = applicationId
	appConfigAdvance.environmentId = environmentId
	return nil
}

func (appConfigAdvance *EnhancedAppConfigAdvance) listConfigurationProfiles() error {
	var nextToken *string = nil
	for {
		input := appconfig.ListConfigurationProfilesInput{
			ApplicationId: aws.String(appConfigAdvance.applicationId),
			NextToken:     nextToken,
		}
		output, err := appConfigAdvance.appConfigClient.ListConfigurationProfiles(&input)
		if err != nil {
			return err
		}
		if len(output.Items) == 0 {
			break
		}

		for _, item := range output.Items {
			configurationProfileNameId.Store(*item.Name, *item.Id)
		}

		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}
	return nil
}

func (appConfigAdvance *EnhancedAppConfigAdvance) listDeploymentStrategies() error {
	var nextToken *string = nil
	for {
		input := appconfig.ListDeploymentStrategiesInput{
			NextToken: nextToken,
		}
		output, err := appConfigAdvance.appConfigClient.ListDeploymentStrategies(&input)
		if err != nil {
			return err
		}
		if len(output.Items) == 0 {
			break
		}

		for _, item := range output.Items {
			deploymentStrategyNameId.Store(*item.Name, *item.Id)
		}

		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}
	return nil
}

func (appConfigAdvance *EnhancedAppConfigAdvance) listApplications() error {
	var nextToken *string = nil
	for {
		input := appconfig.ListApplicationsInput{
			NextToken: nextToken,
		}
		output, err := appConfigAdvance.appConfigClient.ListApplications(&input)
		if err != nil {
			return err
		}
		if len(output.Items) == 0 {
			break
		}

		for _, item := range output.Items {
			applicationNameId[*item.Name] = *item.Id
		}

		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}
	return nil
}

func (appConfigAdvance *EnhancedAppConfigAdvance) listEnvironments(appConfigClient *appconfig.AppConfig, applicationId string) error {
	var nextToken *string = nil
	for {
		input := appconfig.ListEnvironmentsInput{
			ApplicationId: aws.String(applicationId),
			NextToken:     nextToken,
		}
		output, err := appConfigClient.ListEnvironments(&input)
		if err != nil {
			return err
		}
		if len(output.Items) == 0 {
			break
		}

		for _, item := range output.Items {
			environmentNameId[*item.Name] = *item.Id
		}

		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}
	return nil
}

func (appConfigAdvance *EnhancedAppConfigAdvance) ApplyOptions(opts ...Option) error {
	for _, opt := range opts {
		err := opt.apply(appConfigAdvance)
		if err != nil {
			return err
		}
	}
	return nil
}
