package database

import (
	"github.com/alicebob/miniredis/v2"
	"github.com/forceu/gokapi/internal/configuration/database/dbabstraction"
	"github.com/forceu/gokapi/internal/models"
	"github.com/forceu/gokapi/internal/test"
	"log"
	"os"
	"testing"
	"time"
)

var configSqlite = models.DbConnection{
	HostUrl: "./test/gokapi.sqlite",
	Type:    0, // dbabstraction.TypeSqlite
}

var configRedis = models.DbConnection{
	RedisPrefix: "test_",
	HostUrl:     "127.0.0.1:26379",
	Type:        1, // dbabstraction.TypeRedis
}

var mRedis *miniredis.Miniredis

var availableDatabases []dbabstraction.Database

func TestMain(m *testing.M) {

	mRedis = miniredis.NewMiniRedis()
	err := mRedis.StartAddr("127.0.0.1:26379")
	if err != nil {
		log.Fatal("Could not start miniredis")
	}
	exitVal := m.Run()
	mRedis.Close()
	os.RemoveAll("./test/")
	os.Exit(exitVal)
}

func TestInit(t *testing.T) {
	availableDatabases = make([]dbabstraction.Database, 0)
	Connect(configRedis)
	availableDatabases = append(availableDatabases, db)
	Connect(configSqlite)
	availableDatabases = append(availableDatabases, db)
	defer test.ExpectPanic(t)
	Connect(models.DbConnection{Type: 2})
}

func TestApiKeys(t *testing.T) {
	runAllTypesCompareOutput(t, func() any { return GetAllApiKeys() }, map[string]models.ApiKey{})
	newApiKey := models.ApiKey{
		Id:           "test",
		FriendlyName: "testKey",
		LastUsed:     1000,
		Permissions:  10,
	}
	runAllTypesNoOutput(t, func() { SaveApiKey(newApiKey) })
	runAllTypesCompareTwoOutputs(t, func() (any, any) {
		return GetApiKey("test")
	}, newApiKey, true)
	newApiKey.LastUsed = 2000
	runAllTypesNoOutput(t, func() { UpdateTimeApiKey(newApiKey) })
	runAllTypesCompareOutput(t, func() any { return GetAllApiKeys() }, map[string]models.ApiKey{"test": newApiKey})
	runAllTypesNoOutput(t, func() { DeleteApiKey("test") })
	runAllTypesCompareTwoOutputs(t, func() (any, any) {
		return GetApiKey("test")
	}, models.ApiKey{}, false)
}

func TestE2E(t *testing.T) {
	input := models.E2EInfoEncrypted{
		Version:        1,
		Nonce:          []byte("test"),
		Content:        []byte("test2"),
		AvailableFiles: []string{"should", "not", "be", "saved"},
	}
	runAllTypesNoOutput(t, func() { SaveEnd2EndInfo(input) })
	input.AvailableFiles = []string{}
	runAllTypesCompareOutput(t, func() any { return GetEnd2EndInfo() }, input)
	runAllTypesNoOutput(t, func() { DeleteEnd2EndInfo() })
	runAllTypesCompareOutput(t, func() any { return GetEnd2EndInfo() }, models.E2EInfoEncrypted{AvailableFiles: []string{}})
}

func TestSessions(t *testing.T) {
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetSession("newsession") }, models.Session{}, false)
	input := models.Session{
		RenewAt:    time.Now().Add(10 * time.Second).Unix(),
		ValidUntil: time.Now().Add(20 * time.Second).Unix(),
	}
	runAllTypesNoOutput(t, func() { SaveSession("newsession", input) })
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetSession("newsession") }, input, true)
	runAllTypesNoOutput(t, func() { DeleteSession("newsession") })
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetSession("newsession") }, models.Session{}, false)
	runAllTypesNoOutput(t, func() { SaveSession("newsession", input) })
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetSession("newsession") }, input, true)
	runAllTypesNoOutput(t, func() { DeleteAllSessions() })
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetSession("newsession") }, models.Session{}, false)
}

func TestHotlinks(t *testing.T) {
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetHotlink("newhotlink") }, "", false)
	newFile := models.File{Id: "testfile",
		HotlinkId: "newhotlink"}
	runAllTypesNoOutput(t, func() { SaveHotlink(newFile) })
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetHotlink("newhotlink") }, "testfile", true)
	runAllTypesCompareOutput(t, func() any { return GetAllHotlinks() }, []string{"newhotlink"})
	runAllTypesNoOutput(t, func() { DeleteHotlink("newhotlink") })
	runAllTypesCompareOutput(t, func() any { return GetAllHotlinks() }, []string{})
}

func TestMetaData(t *testing.T) {
	runAllTypesCompareOutput(t, func() any { return GetAllMetaDataIds() }, []string{})
	runAllTypesCompareOutput(t, func() any { return GetAllMetadata() }, map[string]models.File{})
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetMetaDataById("testid") }, models.File{}, false)
	file := models.File{
		Id:                 "testid",
		Name:               "Testname",
		Size:               "3Kb",
		SHA1:               "12345556",
		PasswordHash:       "sfffwefwe",
		HotlinkId:          "hotlink",
		ContentType:        "none",
		AwsBucket:          "aws1",
		ExpireAtString:     "In 10 seconds",
		ExpireAt:           time.Now().Add(10 * time.Second).Unix(),
		SizeBytes:          3 * 1024,
		DownloadsRemaining: 2,
		DownloadCount:      5,
		Encryption: models.EncryptionInfo{
			IsEncrypted:         true,
			IsEndToEndEncrypted: true,
			DecryptionKey:       []byte("dekey"),
			Nonce:               []byte("nonce"),
		},
		UnlimitedDownloads: true,
		UnlimitedTime:      true,
	}
	runAllTypesNoOutput(t, func() { SaveMetaData(file) })
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetMetaDataById("testid") }, file, true)
	runAllTypesCompareOutput(t, func() any { return GetAllMetaDataIds() }, []string{"testid"})
	runAllTypesCompareOutput(t, func() any { return GetAllMetadata() }, map[string]models.File{"testid": file})
	runAllTypesNoOutput(t, func() { DeleteMetaData("testid") })
	runAllTypesCompareOutput(t, func() any { return GetAllMetaDataIds() }, []string{})
	runAllTypesCompareOutput(t, func() any { return GetAllMetadata() }, map[string]models.File{})
	runAllTypesCompareTwoOutputs(t, func() (any, any) { return GetMetaDataById("testid") }, models.File{}, false)

	increasedDownload := file
	increasedDownload.DownloadCount = increasedDownload.DownloadCount + 1

	runAllTypesCompareTwoOutputs(t, func() (any, any) {
		SaveMetaData(file)
		IncreaseDownloadCount(file.Id, false)
		return GetMetaDataById(file.Id)
	}, increasedDownload, true)

	increasedDownload.DownloadCount = increasedDownload.DownloadCount + 1
	increasedDownload.DownloadsRemaining = increasedDownload.DownloadsRemaining - 1

	runAllTypesCompareTwoOutputs(t, func() (any, any) {
		IncreaseDownloadCount(file.Id, true)
		return GetMetaDataById(file.Id)
	}, increasedDownload, true)
	runAllTypesNoOutput(t, func() { DeleteMetaData(file.Id) })
}

func TestUpgrade(t *testing.T) {
	runAllTypesNoOutput(t, func() { test.IsEqualBool(t, db.GetDbVersion() != 1, true) })
	runAllTypesNoOutput(t, func() { db.SetDbVersion(1) })
	runAllTypesNoOutput(t, func() { test.IsEqualInt(t, db.GetDbVersion(), 1) })
	runAllTypesNoOutput(t, func() { Upgrade() })
	runAllTypesNoOutput(t, func() { test.IsEqualInt(t, db.GetDbVersion(), db.GetSchemaVersion()) })
}

func TestRunGarbageCollection(t *testing.T) {
	runAllTypesNoOutput(t, func() { RunGarbageCollection() })
}

func TestClose(t *testing.T) {
	runAllTypesNoOutput(t, func() { Close() })
}

func runAllTypesNoOutput(t *testing.T, functionToRun func()) {
	t.Helper()
	for _, database := range availableDatabases {
		db = database
		functionToRun()
	}
}

func runAllTypesCompareOutput(t *testing.T, functionToRun func() any, expectedOutput any) {
	t.Helper()
	for _, database := range availableDatabases {
		db = database
		output := functionToRun()
		test.IsEqual(t, output, expectedOutput)
	}
}

func runAllTypesCompareTwoOutputs(t *testing.T, functionToRun func() (any, any), expectedOutput1, expectedOutput2 any) {
	t.Helper()
	for _, database := range availableDatabases {
		db = database
		output1, output2 := functionToRun()
		test.IsEqual(t, output1, expectedOutput1)
		test.IsEqual(t, output2, expectedOutput2)
	}
}

func TestParseUrl(t *testing.T) {
	expectedOutput := models.DbConnection{}
	output, err := ParseUrl("invalid", false)
	test.IsNotNil(t, err)
	test.IsEqual(t, output, expectedOutput)

	_, err = ParseUrl("", false)
	test.IsNotNil(t, err)
	_, err = ParseUrl("inv\r\nalid", false)
	test.IsNotNil(t, err)
	_, err = ParseUrl("", false)
	test.IsNotNil(t, err)

	expectedOutput = models.DbConnection{
		HostUrl: "./test",
		Type:    dbabstraction.TypeSqlite,
	}
	output, err = ParseUrl("sqlite://./test", false)
	test.IsNil(t, err)
	test.IsEqual(t, output, expectedOutput)

	_, err = ParseUrl("sqlite:///invalid", true)
	test.IsNotNil(t, err)
	output, err = ParseUrl("sqlite:///invalid", false)
	test.IsNil(t, err)
	test.IsEqualString(t, output.HostUrl, "/invalid")

	expectedOutput = models.DbConnection{
		HostUrl:     "127.0.0.1:1234",
		RedisPrefix: "",
		Username:    "",
		Password:    "",
		RedisUseSsl: false,
		Type:        dbabstraction.TypeRedis,
	}
	output, err = ParseUrl("redis://127.0.0.1:1234", false)
	test.IsNil(t, err)
	test.IsEqual(t, output, expectedOutput)

	expectedOutput = models.DbConnection{
		HostUrl:     "127.0.0.1:1234",
		RedisPrefix: "tpref",
		Username:    "tuser",
		Password:    "tpw",
		RedisUseSsl: true,
		Type:        dbabstraction.TypeRedis,
	}
	output, err = ParseUrl("redis://tuser:tpw@127.0.0.1:1234/?ssl=true&prefix=tpref", false)
	test.IsNil(t, err)
	test.IsEqual(t, output, expectedOutput)
}

func TestMigration(t *testing.T) {
	configNew := models.DbConnection{
		RedisPrefix: "testmigrate_",
		HostUrl:     "127.0.0.1:26379",
		Type:        1, // dbabstraction.TypeRedis
	}
	dbOld, err := dbabstraction.GetNew(configSqlite)
	test.IsNil(t, err)
	testFile := models.File{Id: "file1234", HotlinkId: "hotlink123"}
	dbOld.SaveMetaData(testFile)
	dbOld.SaveHotlink(testFile)
	dbOld.SaveApiKey(models.ApiKey{Id: "api123"})
	dbOld.SaveHotlink(testFile)
	dbOld.Close()

	Migrate(configSqlite, configNew)

	dbNew, err := dbabstraction.GetNew(configNew)
	test.IsNil(t, err)
	_, ok := dbNew.GetHotlink("hotlink123")
	test.IsEqualBool(t, ok, true)
	_, ok = dbNew.GetApiKey("api123")
	test.IsEqualBool(t, ok, true)
	_, ok = dbNew.GetMetaDataById("file1234")
	test.IsEqualBool(t, ok, true)
}
