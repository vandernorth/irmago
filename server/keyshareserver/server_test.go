package keyshareserver

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi"
	"github.com/privacybydesign/irmago/internal/keysharecore"
	"github.com/privacybydesign/irmago/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerInvalidMessage(t *testing.T) {
	StartKeyshareServer(t, NewMemoryDatabase(), "")
	defer StopKeyshareServer(t)

	reqData := bytes.NewBufferString("gval;kefsajsdkl;")
	res, err := http.Post("http://localhost:8080/irma_keyshare_server/api/v1/client/register", "application/json", reqData)
	assert.NoError(t, err)
	assert.Equal(t, 400, res.StatusCode)
	res.Body.Close()

	reqData = bytes.NewBufferString("asdlkzdsf;lskajl;kasdjfvl;jzxclvyewr")
	res, err = http.Post("http://localhost:8080/irma_keyshare_server/api/v1/users/verify/pin", "application/json", reqData)
	assert.NoError(t, err)
	assert.Equal(t, 400, res.StatusCode)
	res.Body.Close()

	reqData = bytes.NewBufferString("asdlkzdsf;lskajl;kasdjfvl;jzxclvyewr")
	res, err = http.Post("http://localhost:8080/irma_keyshare_server/api/v1/users/change/pin", "application/json", reqData)
	assert.NoError(t, err)
	assert.Equal(t, 400, res.StatusCode)
	res.Body.Close()

	reqData = bytes.NewBufferString("asdlkzdsf;lskajl;kasdjfvl;jzxclvyewr")
	res, err = http.Post("http://localhost:8080/irma_keyshare_server/api/v1/prove/getCommitments", "application/json", reqData)
	assert.NoError(t, err)
	assert.Equal(t, 400, res.StatusCode)
	res.Body.Close()

	reqData = bytes.NewBufferString("[]")
	res, err = http.Post("http://localhost:8080/irma_keyshare_server/api/v1/prove/getCommitments", "application/json", reqData)
	assert.NoError(t, err)
	assert.Equal(t, 400, res.StatusCode)
	res.Body.Close()

	reqData = bytes.NewBufferString("asdlkzdsf;lskajl;kasdjfvl;jzxclvyewr")
	res, err = http.Post("http://localhost:8080/irma_keyshare_server/api/v1/prove/getResponse", "application/json", reqData)
	assert.NoError(t, err)
	assert.Equal(t, 400, res.StatusCode)
	res.Body.Close()
}

func TestServerHandleValidate(t *testing.T) {
	db := NewMemoryDatabase()
	db.NewUser(KeyshareUserData{
		Username: "",
		Coredata: keysharecore.EncryptedKeysharePacket{},
	})
	var ep keysharecore.EncryptedKeysharePacket
	p, err := base64.StdEncoding.DecodeString("YWJjZB7irkDzwMWtBC6PTItWmO2AgAGm1/gFOyrd+nyt3/0GaHLY5Z1S1TM6N5nzb1Jh+Nqx0z0c3f9R2UyoYuy+pnrerTpYL1mpoZZfz8MPqcrAMsmVdb2kHH0BuAGSC0V28tp1BCVzhYnfMJyrUlNWonsTWSn68Av1BwpIBOGxqBXYfW0JzaffuSmZIyubImmTN7p32ASbseJSNwu0Rg==")
	require.NoError(t, err)
	copy(ep[:], p)
	db.NewUser(KeyshareUserData{
		Username: "testusername",
		Coredata: ep,
	})
	StartKeyshareServer(t, db, "")
	defer StopKeyshareServer(t)

	reqData := bytes.NewBufferString(`{"id":"testusername","pin":"puZGbaLDmFywGhFDi4vW2G87ZhXpaUsvymZwNJfB/SU=\n"}`)
	res, err := http.Post("http://localhost:8080/irma_keyshare_server/api/v1/users/verify/pin", "application/json", reqData)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)
	defer res.Body.Close()
	jwtTxt, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	var jwtMsg keysharePinStatus
	err = json.Unmarshal(jwtTxt, &jwtMsg)
	require.NoError(t, err)
	require.Equal(t, "success", jwtMsg.Status)

	client := &http.Client{}
	req, err := http.NewRequest("POST", "http://localhost:8080/irma_keyshare_server/api/v1/users/isAuthorized", nil)
	req.Header.Add("X-IRMA-Keyshare-Username", "testusername")
	req.Header.Add("Authorization", jwtMsg.Message)
	res, err = client.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, 200, res.StatusCode)
	var msg keyshareAuthorization
	resTxt, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	err = json.Unmarshal(resTxt, &msg)
	assert.NoError(t, err)
	assert.Equal(t, "authorized", msg.Status)

	req, err = http.NewRequest("POST", "http://localhost:8080/irma_keyshare_server/api/v1/users/isAuthorized", nil)
	req.Header.Add("X-IRMA-Keyshare-Username", "testusername")
	req.Header.Add("Authorization", "Bearer "+jwtMsg.Message)
	res, err = client.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, 200, res.StatusCode)
	resTxt, err = ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	err = json.Unmarshal(resTxt, &msg)
	assert.NoError(t, err)
	assert.Equal(t, "authorized", msg.Status)

	req, err = http.NewRequest("POST", "http://localhost:8080/irma_keyshare_server/api/v1/users/isAuthorized", nil)
	req.Header.Add("X-IRMA-Keyshare-Username", "testusername")
	req.Header.Add("Authorization", "eyalksjdf.aljsdklfesdfhas.asdfhasdf")
	res, err = client.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	assert.Equal(t, 200, res.StatusCode)
	resTxt, err = ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	err = json.Unmarshal(resTxt, &msg)
	assert.NoError(t, err)
	assert.Equal(t, "expired", msg.Status)
}

func TestPinTries(t *testing.T) {
	db := NewMemoryDatabase()
	db.NewUser(KeyshareUserData{
		Username: "",
		Coredata: keysharecore.EncryptedKeysharePacket{},
	})
	var ep keysharecore.EncryptedKeysharePacket
	p, err := base64.StdEncoding.DecodeString("YWJjZB7irkDzwMWtBC6PTItWmO2AgAGm1/gFOyrd+nyt3/0GaHLY5Z1S1TM6N5nzb1Jh+Nqx0z0c3f9R2UyoYuy+pnrerTpYL1mpoZZfz8MPqcrAMsmVdb2kHH0BuAGSC0V28tp1BCVzhYnfMJyrUlNWonsTWSn68Av1BwpIBOGxqBXYfW0JzaffuSmZIyubImmTN7p32ASbseJSNwu0Rg==")
	require.NoError(t, err)
	copy(ep[:], p)
	db.NewUser(KeyshareUserData{
		Username: "testusername",
		Coredata: ep,
	})
	StartKeyshareServer(t, &testDB{db: db, ok: true, tries: 1, wait: 0, err: nil}, "")
	defer StopKeyshareServer(t)

	reqData := bytes.NewBufferString(`{"id":"testusername","pin":"puZGbaLDmFywGhFDi4vW2G87Zh"}`)
	res, err := http.Post("http://localhost:8080/irma_keyshare_server/api/v1/users/verify/pin", "application/json", reqData)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)
	defer res.Body.Close()
	jwtTxt, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	var jwtMsg keysharePinStatus
	err = json.Unmarshal(jwtTxt, &jwtMsg)
	require.NoError(t, err)
	require.Equal(t, "failure", jwtMsg.Status)
	require.Equal(t, "1", jwtMsg.Message)

	reqData = bytes.NewBufferString(`{"id":"testusername","oldpin":"puZGbaLDmFywGhFDi4vW2G87Zh","newpin":"ljaksdfj;alkf"}`)
	res, err = http.Post("http://localhost:8080/irma_keyshare_server/api/v1/users/change/pin", "application/json", reqData)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)
	defer res.Body.Close()
	jwtTxt, err = ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	err = json.Unmarshal(jwtTxt, &jwtMsg)
	require.NoError(t, err)
	require.Equal(t, "failure", jwtMsg.Status)
	require.Equal(t, "1", jwtMsg.Message)
}

func TestPinWait(t *testing.T) {
	db := NewMemoryDatabase()
	db.NewUser(KeyshareUserData{
		Username: "",
		Coredata: keysharecore.EncryptedKeysharePacket{},
	})
	var ep keysharecore.EncryptedKeysharePacket
	p, err := base64.StdEncoding.DecodeString("YWJjZB7irkDzwMWtBC6PTItWmO2AgAGm1/gFOyrd+nyt3/0GaHLY5Z1S1TM6N5nzb1Jh+Nqx0z0c3f9R2UyoYuy+pnrerTpYL1mpoZZfz8MPqcrAMsmVdb2kHH0BuAGSC0V28tp1BCVzhYnfMJyrUlNWonsTWSn68Av1BwpIBOGxqBXYfW0JzaffuSmZIyubImmTN7p32ASbseJSNwu0Rg==")
	require.NoError(t, err)
	copy(ep[:], p)
	db.NewUser(KeyshareUserData{
		Username: "testusername",
		Coredata: ep,
	})
	StartKeyshareServer(t, &testDB{db: db, ok: true, tries: 0, wait: 5, err: nil}, "")
	defer StopKeyshareServer(t)

	reqData := bytes.NewBufferString(`{"id":"testusername","pin":"puZGbaLDmFywGhFDi4vW2G87Zh"}`)
	res, err := http.Post("http://localhost:8080/irma_keyshare_server/api/v1/users/verify/pin", "application/json", reqData)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)
	defer res.Body.Close()
	jwtTxt, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	var jwtMsg keysharePinStatus
	err = json.Unmarshal(jwtTxt, &jwtMsg)
	require.NoError(t, err)
	require.Equal(t, "error", jwtMsg.Status)
	require.Equal(t, "5", jwtMsg.Message)

	reqData = bytes.NewBufferString(`{"id":"testusername","oldpin":"puZGbaLDmFywGhFDi4vW2G87Zh","newpin":"ljaksdfj;alkf"}`)
	res, err = http.Post("http://localhost:8080/irma_keyshare_server/api/v1/users/change/pin", "application/json", reqData)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)
	defer res.Body.Close()
	jwtTxt, err = ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	err = json.Unmarshal(jwtTxt, &jwtMsg)
	require.NoError(t, err)
	require.Equal(t, "error", jwtMsg.Status)
	require.Equal(t, "5", jwtMsg.Message)
}

func TestPinWaitRefused(t *testing.T) {
	db := NewMemoryDatabase()
	db.NewUser(KeyshareUserData{
		Username: "",
		Coredata: keysharecore.EncryptedKeysharePacket{},
	})
	var ep keysharecore.EncryptedKeysharePacket
	p, err := base64.StdEncoding.DecodeString("YWJjZB7irkDzwMWtBC6PTItWmO2AgAGm1/gFOyrd+nyt3/0GaHLY5Z1S1TM6N5nzb1Jh+Nqx0z0c3f9R2UyoYuy+pnrerTpYL1mpoZZfz8MPqcrAMsmVdb2kHH0BuAGSC0V28tp1BCVzhYnfMJyrUlNWonsTWSn68Av1BwpIBOGxqBXYfW0JzaffuSmZIyubImmTN7p32ASbseJSNwu0Rg==")
	require.NoError(t, err)
	copy(ep[:], p)
	db.NewUser(KeyshareUserData{
		Username: "testusername",
		Coredata: ep,
	})
	StartKeyshareServer(t, &testDB{db: db, ok: false, tries: 0, wait: 5, err: nil}, "")
	defer StopKeyshareServer(t)

	reqData := bytes.NewBufferString(`{"id":"testusername","pin":"puZGbaLDmFywGhFDi4vW2G87Zh"}`)
	res, err := http.Post("http://localhost:8080/irma_keyshare_server/api/v1/users/verify/pin", "application/json", reqData)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)
	defer res.Body.Close()
	jwtTxt, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	var jwtMsg keysharePinStatus
	err = json.Unmarshal(jwtTxt, &jwtMsg)
	require.NoError(t, err)
	require.Equal(t, "error", jwtMsg.Status)
	require.Equal(t, "5", jwtMsg.Message)

	reqData = bytes.NewBufferString(`{"id":"testusername","oldpin":"puZGbaLDmFywGhFDi4vW2G87Zh","newpin":"ljaksdfj;alkf"}`)
	res, err = http.Post("http://localhost:8080/irma_keyshare_server/api/v1/users/change/pin", "application/json", reqData)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)
	defer res.Body.Close()
	jwtTxt, err = ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	err = json.Unmarshal(jwtTxt, &jwtMsg)
	require.NoError(t, err)
	require.Equal(t, "error", jwtMsg.Status)
	require.Equal(t, "5", jwtMsg.Message)
}

func TestMissingUser(t *testing.T) {
	StartKeyshareServer(t, NewMemoryDatabase(), "")
	defer StopKeyshareServer(t)

	client := &http.Client{}
	req, err := http.NewRequest("POST", "http://localhost:8080/irma_keyshare_server/api/v1/users/isAuthorized", nil)
	require.NoError(t, err)
	req.Header.Add("X-IRMA-Keyshare-Username", "doesnotexist")
	req.Header.Add("Authorization", "ey.ey.ey")
	res, err := client.Do(req)
	assert.NoError(t, err)
	assert.NotEqual(t, 200, res.StatusCode)
	res.Body.Close()

	reqData := bytes.NewBufferString(`{"id":"doesnotexist","pin":"bla"}`)
	res, err = http.Post("http://localhost:8080/irma_keyshare_server/api/v1/users/verify/pin", "application/json", reqData)
	assert.NoError(t, err)
	assert.NotEqual(t, 200, res.StatusCode)
	res.Body.Close()

	reqData = bytes.NewBufferString(`{"id":"doesnotexist","oldpin":"old","newpin":"new"}`)
	res, err = http.Post("http://localhost:8080/irma_keyshare_server/api/v1/users/change/pin", "application/json", reqData)
	assert.NoError(t, err)
	assert.NotEqual(t, 200, res.StatusCode)
	res.Body.Close()

	reqData = bytes.NewBufferString(`["test.test-3"]`)
	req, err = http.NewRequest("POST", "http://localhost:8080/irma_keyshare_server/api/v1/prove/getCommitments", reqData)
	require.NoError(t, err)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-IRMA-Keyshare-Username", "doesnotexist")
	req.Header.Add("Authorization", "ey.ey.ey")
	res, err = client.Do(req)
	assert.NoError(t, err)
	assert.NotEqual(t, 200, res.StatusCode)
	res.Body.Close()

	reqData = bytes.NewBufferString("123456789")
	req, err = http.NewRequest("POST", "http://localhost:8080/irma_keyshare_server/api/v1/prove/getResponse", reqData)
	require.NoError(t, err)
	req.Header.Add("X-IRMA-Keyshare-Username", "doesnotexist")
	req.Header.Add("Authorization", "ey.ey.ey")
	res, err = client.Do(req)
	assert.NoError(t, err)
	assert.NotEqual(t, 200, res.StatusCode)
	res.Body.Close()
}

func TestInvalidKeyshareSessions(t *testing.T) {
	db := NewMemoryDatabase()
	db.NewUser(KeyshareUserData{
		Username: "",
		Coredata: keysharecore.EncryptedKeysharePacket{},
	})
	var ep keysharecore.EncryptedKeysharePacket
	p, err := base64.StdEncoding.DecodeString("YWJjZB7irkDzwMWtBC6PTItWmO2AgAGm1/gFOyrd+nyt3/0GaHLY5Z1S1TM6N5nzb1Jh+Nqx0z0c3f9R2UyoYuy+pnrerTpYL1mpoZZfz8MPqcrAMsmVdb2kHH0BuAGSC0V28tp1BCVzhYnfMJyrUlNWonsTWSn68Av1BwpIBOGxqBXYfW0JzaffuSmZIyubImmTN7p32ASbseJSNwu0Rg==")
	require.NoError(t, err)
	copy(ep[:], p)
	db.NewUser(KeyshareUserData{
		Username: "testusername",
		Coredata: ep,
	})
	StartKeyshareServer(t, db, "")
	defer StopKeyshareServer(t)

	reqData := bytes.NewBufferString(`{"id":"testusername","pin":"puZGbaLDmFywGhFDi4vW2G87ZhXpaUsvymZwNJfB/SU=\n"}`)
	res, err := http.Post("http://localhost:8080/irma_keyshare_server/api/v1/users/verify/pin", "application/json", reqData)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)
	defer res.Body.Close()
	jwtTxt, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)
	var jwtMsg keysharePinStatus
	err = json.Unmarshal(jwtTxt, &jwtMsg)
	require.NoError(t, err)
	require.Equal(t, "success", jwtMsg.Status)

	client := &http.Client{}

	reqData = bytes.NewBufferString("12345678")
	req, err := http.NewRequest("POST", "http://localhost:8080/irma_keyshare_server/api/v1/prove/getResponse", reqData)
	require.NoError(t, err)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-IRMA-Keyshare-Username", "testusername")
	req.Header.Add("Authorization", "Bearer "+jwtMsg.Message)
	res, err = client.Do(req)
	assert.NoError(t, err)
	assert.NotEqual(t, 200, res.StatusCode)
	res.Body.Close()

	reqData = bytes.NewBufferString(`["test.test-3"]`)
	req, err = http.NewRequest("POST", "http://localhost:8080/irma_keyshare_server/api/v1/prove/getCommitments", reqData)
	require.NoError(t, err)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-IRMA-Keyshare-Username", "testusername")
	req.Header.Add("Authorization", "fakeauthorization")
	res, err = client.Do(req)
	assert.NoError(t, err)
	assert.NotEqual(t, 200, res.StatusCode)
	res.Body.Close()

	reqData = bytes.NewBufferString(`["test.test-3"]`)
	req, err = http.NewRequest("POST", "http://localhost:8080/irma_keyshare_server/api/v1/prove/getCommitments", reqData)
	require.NoError(t, err)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-IRMA-Keyshare-Username", "testusername")
	req.Header.Add("Authorization", "Bearer "+jwtMsg.Message)
	res, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)
	res.Body.Close()

	reqData = bytes.NewBufferString("12345678")
	req, err = http.NewRequest("POST", "http://localhost:8080/irma_keyshare_server/api/v1/prove/getResponse", reqData)
	require.NoError(t, err)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-IRMA-Keyshare-Username", "testusername")
	req.Header.Add("Authorization", "fakeauthorization")
	res, err = client.Do(req)
	assert.NoError(t, err)
	assert.NotEqual(t, 200, res.StatusCode)
	res.Body.Close()

	reqData = bytes.NewBufferString(`["test.test-3"]`)
	req, err = http.NewRequest("POST", "http://localhost:8080/irma_keyshare_server/api/v1/prove/getCommitments", reqData)
	require.NoError(t, err)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-IRMA-Keyshare-Username", "testusername")
	req.Header.Add("Authorization", "Bearer "+jwtMsg.Message)
	res, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, res.StatusCode)
	res.Body.Close()

	reqData = bytes.NewBufferString("12345678")
	req, err = http.NewRequest("POST", "http://localhost:8080/irma_keyshare_server/api/v1/prove/getResponse", reqData)
	require.NoError(t, err)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-IRMA-Keyshare-Username", "testusername")
	req.Header.Add("Authorization", "Bearer "+jwtMsg.Message)
	res, err = client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, res.StatusCode)
	res.Body.Close()
}

var keyshareServ *http.Server

func StartKeyshareServer(t *testing.T, db KeyshareDB, emailserver string) {
	testdataPath := test.FindTestdataFolder(t)
	s, err := New(&Configuration{
		SchemesPath:           filepath.Join(testdataPath, "irma_configuration"),
		URL:                   "http://localhost:8080/irma_keyshare_server/",
		DB:                    db,
		JwtKeyId:              0,
		JwtPrivateKeyFile:     filepath.Join(testdataPath, "jwtkeys", "kss-sk.pem"),
		StoragePrimaryKeyFile: filepath.Join(testdataPath, "keyshareStorageTestkey"),
		KeyshareCredential:    "test.test.mijnirma",
		KeyshareAttribute:     "email",
		EmailServer:           emailserver,
		EmailFrom:             "test@example.com",
		DefaultLanguage:       "en",
		RegistrationEmailFiles: map[string]string{
			"en": filepath.Join(testdataPath, "emailtemplate.html"),
		},
		RegistrationEmailSubject: map[string]string{
			"en": "testsubject",
		},
		VerificationURL: map[string]string{
			"en": "http://example.com/verify/",
		},
	})
	require.NoError(t, err)

	r := chi.NewRouter()
	r.Mount("/irma_keyshare_server/", s.Handler())

	keyshareServ = &http.Server{
		Addr:    "localhost:8080",
		Handler: r,
	}

	go func() {
		err := keyshareServ.ListenAndServe()
		if err == http.ErrServerClosed {
			err = nil
		}
		assert.NoError(t, err)
	}()
}

func StopKeyshareServer(t *testing.T) {
	err := keyshareServ.Shutdown(context.Background())
	assert.NoError(t, err)
}

type testDB struct {
	db    KeyshareDB
	ok    bool
	tries int
	wait  int64
	err   error
}

func (db *testDB) NewUser(user KeyshareUserData) (KeyshareUser, error) {
	return db.db.NewUser(user)
}

func (db *testDB) User(username string) (KeyshareUser, error) {
	return db.db.User(username)
}

func (db *testDB) UpdateUser(user KeyshareUser) error {
	return db.db.UpdateUser(user)
}

func (db *testDB) ReservePincheck(user KeyshareUser) (bool, int, int64, error) {
	return db.ok, db.tries, db.wait, db.err
}

func (db *testDB) ClearPincheck(user KeyshareUser) error {
	return db.db.ClearPincheck(user)
}

func (db *testDB) SetSeen(user KeyshareUser) error {
	return db.db.SetSeen(user)
}

func (db *testDB) AddLog(user KeyshareUser, entrytype LogEntryType, params interface{}) error {
	return db.db.AddLog(user, entrytype, params)
}

func (db *testDB) AddEmailVerification(user KeyshareUser, email, token string) error {
	return db.db.AddEmailVerification(user, email, token)
}
