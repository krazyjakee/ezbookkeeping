package datastore

import (
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mayswind/ezbookkeeping/pkg/errs"
	"github.com/mayswind/ezbookkeeping/pkg/settings"
)

func TestGetMysqlConnectionString_TCP(t *testing.T) {
	expectedValue := "username:password@tcp(1.2.3.4:3306)/dbname?charset=utf8mb4&parseTime=true"
	actualValue, err := getMysqlConnectionString(&settings.DatabaseConfig{
		DatabaseType:     "mysql",
		DatabaseHost:     "1.2.3.4:3306",
		DatabaseName:     "dbname",
		DatabaseUser:     "username",
		DatabasePassword: "password",
	})

	assert.Nil(t, err)
	assert.Equal(t, expectedValue, actualValue)
}

func TestGetMysqlConnectionString_UnixSocket(t *testing.T) {
	expectedValue := "username:password@unix(/path/to/mysql.sock)/dbname?charset=utf8mb4&parseTime=true"
	actualValue, err := getMysqlConnectionString(&settings.DatabaseConfig{
		DatabaseType:     "mysql",
		DatabaseHost:     "/path/to/mysql.sock",
		DatabaseName:     "dbname",
		DatabaseUser:     "username",
		DatabasePassword: "password",
	})

	assert.Nil(t, err)
	assert.Equal(t, expectedValue, actualValue)
}

func TestGetPostgreSQLConnectionString_TCP(t *testing.T) {
	expectedValue := "postgres://username:password@1.2.3.4:5432/dbname?sslmode=disable"
	actualValue, err := getPostgresConnectionString(&settings.DatabaseConfig{
		DatabaseType:     "postgres",
		DatabaseHost:     "1.2.3.4:5432",
		DatabaseName:     "dbname",
		DatabaseUser:     "username",
		DatabasePassword: "password",
		DatabaseSSLMode:  "disable",
	})

	assert.Nil(t, err)
	assert.Equal(t, expectedValue, actualValue)
}

func TestGetPostgreSQLConnectionString_UnixSocket(t *testing.T) {
	expectedValue := "postgres:///dbname?sslmode=disable&host=/path/to/postgres.sock&user=username&password=password"
	actualValue, err := getPostgresConnectionString(&settings.DatabaseConfig{
		DatabaseType:     "postgres",
		DatabaseHost:     "/path/to/postgres.sock",
		DatabaseName:     "dbname",
		DatabaseUser:     "username",
		DatabasePassword: "password",
		DatabaseSSLMode:  "disable",
	})

	assert.Nil(t, err)
	assert.Equal(t, expectedValue, actualValue)
}

func TestGetPostgreSQLConnectionString_TCPRoundTrip(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		host     string
		username string
		password string
	}{
		{name: "spaces", host: "localhost:5432", username: "book keeper", password: "secret phrase"},
		{name: "plus signs", host: "localhost:5432", username: "book+keeper", password: "secret+phrase"},
		{name: "reserved characters", host: "localhost:5432", username: "book@keeper:%", password: "secret@phrase:%"},
		{name: "IPv6", host: "[::1]:5432", username: "username", password: "password"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			connectionString, err := getPostgresConnectionString(&settings.DatabaseConfig{
				DatabaseType:     "postgres",
				DatabaseHost:     testCase.host,
				DatabaseName:     "dbname",
				DatabaseUser:     testCase.username,
				DatabasePassword: testCase.password,
				DatabaseSSLMode:  "disable",
			})
			require.NoError(t, err)

			parsed, err := url.Parse(connectionString)
			require.NoError(t, err)
			require.NotNil(t, parsed.User)
			assert.Equal(t, testCase.username, parsed.User.Username())
			password, hasPassword := parsed.User.Password()
			assert.True(t, hasPassword)
			assert.Equal(t, testCase.password, password)
			assert.Equal(t, testCase.host, parsed.Host)
			_, port, err := net.SplitHostPort(parsed.Host)
			require.NoError(t, err)
			assert.Equal(t, "5432", port)
			assert.Equal(t, "/dbname", parsed.Path)
			assert.Equal(t, "disable", parsed.Query().Get("sslmode"))
		})
	}
}

func TestGetPostgreSQLConnectionString_InvalidHost(t *testing.T) {
	for _, host := range []string{"localhost", "[::1", "::1:5432"} {
		t.Run(host, func(t *testing.T) {
			connectionString, err := getPostgresConnectionString(&settings.DatabaseConfig{
				DatabaseHost: host,
			})
			assert.ErrorIs(t, err, errs.ErrDatabaseHostInvalid)
			assert.Empty(t, connectionString)
		})
	}
}
