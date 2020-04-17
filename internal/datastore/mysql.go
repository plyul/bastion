package datastore

import (
	"bastion/internal/api"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql" // use MySQL implementation of database/sql interface
	"sync"
)

type mysqlDriver struct {
	// TODO: https://www.reddit.com/r/golang/comments/6wll4z/lots_of_prepared_statements_how_do_i_deal_with/
	db                        *sql.DB
	userStmt                  *sql.Stmt
	protocolsStmt             *sql.Stmt
	networksStmt              *sql.Stmt
	networkByMandateIDStmt    *sql.Stmt
	mandatesStmt              *sql.Stmt
	credentialsStmt           *sql.Stmt
	createSessionTemplateStmt *sql.Stmt
	sessionTemplatesStmt      *sql.Stmt
	deleteSessionTemplateStmt *sql.Stmt
	createSessionStmt         *sql.Stmt
	sessionStmt               *sql.Stmt
	deleteSessionStmt         *sql.Stmt
}

var openDbOnce sync.Once
var openDbOnceError error
var instance mysqlDriver

func storageInstance() (*mysqlDriver, error) {
	// TODO: Реализовать логику ожидания доступности БД (+ уведомление readinessProbe)
	var err error
	openDbOnce.Do(func() {
		instance.db, err = sql.Open("mysql", config.DataSourceName)
		if err != nil {
			config.Logger.Error(err.Error())
			openDbOnceError = err
			return
		}
		err = instance.db.Ping()
		if err != nil {
			config.Logger.Error(err.Error())
			openDbOnceError = err
			return
		}
		err = prepareStatements()
		if err != nil {
			openDbOnceError = err
		}
	})
	if openDbOnceError != nil {
		return nil, openDbOnceError
	}
	if instance.db == nil {
		return nil, err
	}
	err = instance.db.Ping()
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func prepareStatements() error {
	var err error

	instance.protocolsStmt, err = instance.db.Prepare("SELECT pk, name, default_port FROM protocols")
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}

	instance.networksStmt, err = instance.db.Prepare("SELECT pk, name, endpoint, servicepoint " +
		"FROM networks")
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}

	instance.networkByMandateIDStmt, err = instance.db.Prepare("SELECT n.pk, n.name, n.endpoint, n.servicepoint " +
		"FROM networks as n " +
		"LEFT JOIN mandates as m ON n.pk = m.network_id " +
		"WHERE m.pk=?")
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}

	instance.userStmt, err = instance.db.Prepare("SELECT pk, name " +
		"FROM users " +
		"WHERE name=?")
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}

	instance.mandatesStmt, err = instance.db.Prepare("SELECT m.pk, " +
		"m.name " +
		"FROM users_mandates as um " +
		"INNER JOIN mandates as m ON um.mandate_id = m.pk " +
		"WHERE um.user_id IN (SELECT pk FROM users WHERE name=?)")
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}

	instance.credentialsStmt, err = instance.db.Prepare("SELECT " +
		"target_login, target_password, target_private_key " +
		"FROM target_credentials " +
		"WHERE pk=?")
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}

	instance.createSessionTemplateStmt, err = instance.db.Prepare("INSERT INTO session_templates" +
		"(name, user_id, target_proto_id, target_host, target_port, mandate_id, " +
		"custom_target_network_id, custom_target_login, custom_target_password, custom_target_private_key) " +
		"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}

	instance.sessionTemplatesStmt, err = instance.db.Prepare("SELECT st.pk, " +
		"st.name, st.target_proto_id, st.target_host, st.target_port, st.mandate_id, st.custom_target_network_id, st.custom_target_login, st.custom_target_password, st.custom_target_private_key " +
		"FROM session_templates as st " +
		"WHERE st.user_id IN (SELECT pk FROM users WHERE name=?)" +
		"ORDER BY st.name")
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}

	instance.deleteSessionTemplateStmt, err = instance.db.Prepare("DELETE " +
		"FROM session_templates " +
		"WHERE user_id IN (SELECT pk FROM users WHERE name=?) AND pk=?")
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}

	instance.createSessionStmt, err = instance.db.Prepare("INSERT INTO sessions " +
		"(token, origin_ip, user_id, target_proto_id, target_host, target_port, mandate_id, " +
		"custom_target_network_id, custom_target_login, custom_target_password, custom_target_private_key) " +
		"VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}

	instance.sessionStmt, err = instance.db.Prepare("SELECT " +
		"s.origin_ip, p.name, s.target_host, s.target_port, s.mandate_id, " +
		"s.custom_target_network_id, s.custom_target_login, s.custom_target_password, s.custom_target_private_key " +
		"FROM sessions as s " +
		"LEFT JOIN protocols p on s.target_proto_id = p.pk " +
		"WHERE s.token=?")
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}

	instance.deleteSessionStmt, err = instance.db.Prepare("DELETE " +
		"FROM sessions " +
		"WHERE sessions.token=?")
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}

	return nil
}

func Open() error {
	_, err := storageInstance()
	if err != nil {
		return err
	}
	return nil
}

func Close() error {
	storage, err := storageInstance()
	if err != nil {
		return err
	}
	err = storage.userStmt.Close()
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	err = storage.protocolsStmt.Close()
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	err = storage.networksStmt.Close()
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	err = storage.networkByMandateIDStmt.Close()
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	err = storage.mandatesStmt.Close()
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	err = storage.credentialsStmt.Close()
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	err = storage.createSessionTemplateStmt.Close()
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	err = storage.sessionTemplatesStmt.Close()
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	err = storage.deleteSessionTemplateStmt.Close()
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	err = storage.createSessionStmt.Close()
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	err = storage.sessionStmt.Close()
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	err = storage.deleteSessionStmt.Close()
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	err = storage.db.Close()
	storage.db = nil
	return err
}

func User(userName string) (api.User, error) {
	storage, err := storageInstance()
	if err != nil {
		return api.User{}, err
	}

	row := storage.userStmt.QueryRow(userName)
	var user api.User
	err = row.Scan(&user.ID, &user.Name)
	if err != nil {
		config.Logger.Error(err.Error())
		return api.User{}, err
	}
	return user, nil
}

func Protocols() ([]api.Protocol, error) {
	storage, err := storageInstance()
	if err != nil {
		return nil, err
	}
	var protos []api.Protocol
	rows, err := storage.protocolsStmt.Query()
	if err != nil {
		config.Logger.Error(err.Error())
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var proto api.Protocol
		err := rows.Scan(&proto.ID, &proto.Name, &proto.DefaultPort)
		if err != nil {
			config.Logger.Error(err.Error())
			return nil, err
		}
		protos = append(protos, proto)
	}
	return protos, nil
}

func Networks() ([]api.Network, error) {
	storage, err := storageInstance()
	if err != nil {
		return nil, err
	}
	var networks []api.Network
	rows, err := storage.networksStmt.Query()
	if err != nil {
		config.Logger.Error(err.Error())
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var network api.Network
		err := rows.Scan(&network.ID, &network.Name, &network.Endpoint, &network.Servicepoint)
		if err != nil {
			config.Logger.Error(err.Error())
			return nil, err
		}
		networks = append(networks, network)
	}
	return networks, nil
}

func NetworkByID(id int) (api.Network, error) {
	result := api.Network{}
	networks, err := Networks()
	if err != nil {
		return result, err
	}
	for _, n := range networks {
		if n.ID == id {
			result = n
			break
		}
	}
	return result, nil
}

func NetworkByMandateID(mandateID int) (api.Network, error) {
	network := api.Network{}
	storage, err := storageInstance()
	if err != nil {
		return network, err
	}
	row := storage.networkByMandateIDStmt.QueryRow(mandateID)
	err = row.Scan(&network.ID, &network.Name, &network.Endpoint, &network.Servicepoint)
	if err != nil {
		config.Logger.Error(err.Error())
		return network, err
	}
	return network, nil
}

func Mandates(userName string) ([]api.Mandate, error) {
	storage, err := storageInstance()
	if err != nil {
		return nil, err
	}
	var mandates []api.Mandate
	rows, err := storage.mandatesStmt.Query(userName)
	if err != nil {
		config.Logger.Error(err.Error())
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var mandate api.Mandate
		err := rows.Scan(&mandate.ID, &mandate.Name)
		if err != nil {
			config.Logger.Error(err.Error())
			return nil, err
		}
		mandates = append(mandates, mandate)
	}
	return mandates, nil
}

func CreateSessionTemplate(userName string, st api.SessionTemplate) error {
	storage, err := storageInstance()
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	user, err := User(userName)
	if err != nil {
		return err
	}
	mandateID := sql.NullInt32{Int32: int32(st.MandateID), Valid: false}
	customTargetNetworkID := sql.NullInt32{Int32: int32(st.CustomTargetNetworkID), Valid: false}
	customTargetLogin := sql.NullString{String: st.CustomTargetLogin, Valid: false}
	customTargetPassword := sql.NullString{String: st.CustomTargetPassword, Valid: false}
	customTargetPrivKey := sql.NullString{String: st.CustomTargetPrivKey, Valid: false}
	switch st.AccessType {
	case api.AccessTypeMandate:
		mandateID.Valid = true
	case api.AccessTypeCustom:
		customTargetNetworkID.Valid = true
		customTargetLogin.Valid = true
		customTargetPassword.Valid = true
		customTargetPrivKey.Valid = true
	default:
		err := errors.New("unknown access type")
		config.Logger.Error(err.Error())
		return err
	}
	result, err := storage.createSessionTemplateStmt.Exec(
		st.Name,
		user.ID,
		st.TargetProtocolID,
		st.TargetHost,
		st.TargetPort,
		mandateID,
		customTargetNetworkID,
		customTargetLogin,
		customTargetPassword,
		customTargetPrivKey)
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	ra, err := result.RowsAffected()
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	if ra != 1 {
		err := fmt.Errorf("unexpected number of rows (%d) was affected, expect one", ra)
		config.Logger.Error(err.Error())
		return err
	}
	return nil
}

func SessionTemlates(userName string) ([]api.SessionTemplate, error) {
	storage, err := storageInstance()
	if err != nil {
		return nil, err
	}
	var sessionTemplates []api.SessionTemplate
	rows, err := storage.sessionTemplatesStmt.Query(userName)
	if err != nil {
		config.Logger.Error(err.Error())
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var st api.SessionTemplate
		var mandateID sql.NullInt64
		var targetNetwork sql.NullInt64
		var targetLogin sql.NullString
		var targetPassword sql.NullString
		var targetPrivKey sql.NullString
		err := rows.Scan(
			&st.ID,
			&st.Name,
			&st.TargetProtocolID,
			&st.TargetHost,
			&st.TargetPort,
			&mandateID,
			&targetNetwork,
			&targetLogin,
			&targetPassword,
			&targetPrivKey)
		if err != nil {
			config.Logger.Error(err.Error())
			return nil, err
		}

		if mandateID.Valid {
			st.MandateID = int(mandateID.Int64)
		} else {
			st.CustomTargetNetworkID = int(targetNetwork.Int64)
			st.CustomTargetLogin = targetLogin.String
			st.CustomTargetPassword = targetPassword.String
			st.CustomTargetPrivKey = targetPrivKey.String
		}
		sessionTemplates = append(sessionTemplates, st)
	}
	return sessionTemplates, nil
}

func DeleteSessionTemplate(userName string, id int) error {
	storage, err := storageInstance()
	if err != nil {
		return err
	}
	result, err := storage.deleteSessionTemplateStmt.Exec(userName, id)
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	if n != int64(1) {
		err := errors.New("wrong number of affected rows")
		config.Logger.Warn(err.Error())
		return err
	}
	return nil
}

func CreateSession(sessionToken string, sess api.CreateSessionDTO) error {
	storage, err := storageInstance()
	if err != nil {
		return err
	}
	user, err := User(sess.UserName)
	if err != nil {
		return err
	}
	mandateID := sql.NullInt32{Int32: int32(sess.MandateID), Valid: false}
	customTargetNetworkID := sql.NullInt32{Int32: int32(sess.CustomTargetNetworkID), Valid: false}
	customTargetLogin := sql.NullString{String: sess.CustomTargetLogin, Valid: false}
	customTargetPassword := sql.NullString{String: sess.CustomTargetPassword, Valid: false}
	customTargetPrivKey := sql.NullString{String: sess.CustomTargetPrivKey, Valid: false}
	switch sess.AccessType {
	case api.AccessTypeMandate:
		mandateID.Valid = true
	case api.AccessTypeCustom:
		customTargetNetworkID.Valid = true
		customTargetLogin.Valid = true
		customTargetPassword.Valid = true
		customTargetPrivKey.Valid = true
	default:
		err := errors.New("unknown access type")
		config.Logger.Error(err.Error())
		return err
	}
	result, err := storage.createSessionStmt.Exec(
		sessionToken,
		sess.OriginIP,
		user.ID,
		sess.TargetProtocolID,
		sess.TargetHost,
		sess.TargetPort,
		mandateID,
		customTargetNetworkID,
		customTargetLogin,
		customTargetPassword,
		customTargetPrivKey)
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	ra, err := result.RowsAffected()
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	if ra != 1 {
		err := fmt.Errorf("unexpected number of rows (%d) was affected, expect one", ra)
		config.Logger.Error(err.Error())
		return err
	}
	return nil
}

func Session(sessionToken string) (api.ReadSessionDTO, error) {
	result := api.ReadSessionDTO{}
	storage, err := storageInstance()
	if err != nil {
		return result, err
	}

	var session api.ReadSessionDTO
	var mandateID sql.NullInt64
	var targetNetwork sql.NullInt64
	var targetLogin sql.NullString
	var targetPassword sql.NullString
	var targetPrivKey sql.NullString
	row := storage.sessionStmt.QueryRow(sessionToken)
	err = row.Scan(
		&session.OriginIP,
		&session.TargetProtocol,
		&session.TargetHost,
		&session.TargetPort,
		&mandateID,
		&targetNetwork,
		&targetLogin,
		&targetPassword,
		&targetPrivKey)
	if err != nil {
		config.Logger.Error(err.Error())
		return result, err
	}

	if mandateID.Valid {
		network, err := NetworkByMandateID(int(mandateID.Int64))
		if err != nil {
			return result, err
		}
		session.TargetNetwork = network.Name

		row = storage.credentialsStmt.QueryRow(mandateID)
		err = row.Scan(
			&session.TargetLogin,
			&targetPassword,
			&targetPrivKey)
		if err != nil {
			config.Logger.Error(err.Error())
			return api.ReadSessionDTO{}, err
		}
		if targetPrivKey.Valid { // Авторизация по ключу имеет приоритет перед парольной
			session.TargetPrivKey = targetPrivKey.String
			session.TargetPassword = ""
		} else if targetPassword.Valid {
			session.TargetPassword = targetPassword.String
			session.TargetPrivKey = ""
		}
	} else {
		network, _ := NetworkByID(int(targetNetwork.Int64))
		session.TargetNetwork = network.Name
		session.TargetLogin = targetLogin.String
		session.TargetPassword = targetPassword.String
		session.TargetPrivKey = targetPrivKey.String
	}
	return session, nil
}

func DeleteSession(token string) error {
	storage, err := storageInstance()
	if err != nil {
		return err
	}
	_, err = storage.deleteSessionStmt.Exec(token)
	if err != nil {
		config.Logger.Error(err.Error())
		return err
	}
	return nil
}
