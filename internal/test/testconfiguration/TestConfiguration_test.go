package testconfiguration

import (
	"Gokapi/internal/helper"
	"Gokapi/internal/storage/aws"
	"Gokapi/internal/test"
	"os"
	"testing"
)

func TestCreate(t *testing.T) {
	Create(true)
	test.IsEqualBool(t, helper.FolderExists(dataDir), true)
	test.IsEqualBool(t, helper.FileExists(configFile), true)
	test.IsEqualBool(t, helper.FileExists("test/data/a8fdc205a9f19cc1c7507a60c4f01b13d11d7fd0"), true)
}

func TestDelete(t *testing.T) {
	Delete()
	test.IsEqualBool(t, helper.FolderExists(dataDir), false)
}

func TestMockInputStdin(t *testing.T) {
	original := StartMockInputStdin(dataDir)
	result := helper.ReadLine()
	StopMockInputStdin(original)
	test.IsEqualString(t, result, dataDir)
}

func TestSetUpgradeConfigFile(t *testing.T) {
	os.Remove(configFile)
	WriteUpgradeConfigFile()
	test.IsEqualBool(t, helper.FileExists(configFile), true)
	TestDelete(t)
}

func TestEnableS3(t *testing.T) {
	EnableS3()
	if aws.IsMockApi {
		test.IsEqualString(t, os.Getenv("AWS_REGION"), "mock-region-1")
	}
}
func TestDisableS3S3(t *testing.T) {
	DisableS3()
	if aws.IsMockApi {
		test.IsEqualString(t, os.Getenv("AWS_REGION"), "")
	}
}

func TestWriteSslCertificates(t *testing.T) {
	test.IsEqualBool(t, helper.FileExists("test/ssl.key"), false)
	WriteSslCertificates(true)
	test.IsEqualBool(t, helper.FileExists("test/ssl.key"), true)
	os.Remove("test/ssl.key")
	test.IsEqualBool(t, helper.FileExists("test/ssl.key"), false)
	WriteSslCertificates(false)
	test.IsEqualBool(t, helper.FileExists("test/ssl.key"), true)
	Delete()
}
