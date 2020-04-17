package server

import (
	"bastion/internal/api"
	"bastion/internal/datastore"
	"errors"
	"github.com/labstack/echo/v4"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/zap"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

func (app *BastionServer) authCallback(context echo.Context) error {
	rl := context.Get(requestLoggerContextKey).(*zap.Logger)
	params, err := context.FormParams()
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}

	session, err := app.sessions.Get(context.Request(), cookieName)
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}
	stateTokenExpected := session.Values["stateToken"]
	stateTokenGiven := params.Get("state")
	if stateTokenExpected != stateTokenGiven {
		err := errors.New("state verification failed")
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}

	token, err := app.oidcClient.FetchToken(params.Get("code"), rl)
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}
	err = app.saveTokenToSession(token, context)
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}
	return context.Redirect(http.StatusFound, "/app/main")
}

func (app *BastionServer) logoutHandler(context echo.Context) error {
	return context.NoContent(http.StatusNotImplemented)
}

func (app *BastionServer) indexHandler(context echo.Context) error {
	rl := context.Get(requestLoggerContextKey).(*zap.Logger)
	userName, ok := context.Get("SID").(string)
	if !ok {
		rl.Error("unable to get SID from request context")
		return context.NoContent(http.StatusInternalServerError)
	}
	var err error
	type indexTemplateData struct {
		DisplayName string
		Email       string
		Protocols   []api.Protocol
		Mandates    []api.Mandate
		Networks    []api.Network
	}
	data := indexTemplateData{}
	data.DisplayName = context.Get("DisplayName").(string)
	data.Email = context.Get("Email").(string)
	data.Mandates, err = datastore.Mandates(userName)
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}
	data.Protocols, err = datastore.Protocols()
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}
	data.Networks, err = datastore.Networks()
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}
	return context.Render(http.StatusOK, "index", data)
}

func (app *BastionServer) createSessionHandler(context echo.Context) error {
	rl := context.Get(requestLoggerContextKey).(*zap.Logger)
	userName, ok := context.Get("SID").(string)
	if !ok {
		rl.Error("unable to get SID from request context")
		return context.NoContent(http.StatusInternalServerError)
	}
	params, err := context.FormParams()
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}
	for param := range params {
		rl.Debug("CreateSession data dump", zap.String("key", param), zap.String("value", params.Get(param)))
	}

	targetProtocolID, err := strconv.Atoi(params.Get("protocol_id"))
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}
	targetPort, err := strconv.Atoi(params.Get("port"))
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}

	var mandateID, networkID int
	targetNetwork := api.Network{}
	accessType := api.AccessType(params.Get("access_type"))
	switch accessType {
	case api.AccessTypeMandate:
		mandateID, err = strconv.Atoi(params.Get("mandate_id"))
		if err != nil {
			rl.Error(err.Error())
			return context.NoContent(http.StatusInternalServerError)
		}
		err = checkMandate(userName, mandateID)
		if err != nil {
			rl.Error(err.Error())
			return context.NoContent(http.StatusInternalServerError)
		}
		targetNetwork, err = datastore.NetworkByMandateID(mandateID)
		if err != nil {
			rl.Error(err.Error())
			return context.NoContent(http.StatusInternalServerError)
		}
	case api.AccessTypeCustom:
		networkID, err = strconv.Atoi(params.Get("custom_network_id"))
		if err != nil {
			rl.Error(err.Error())
			return context.NoContent(http.StatusInternalServerError)
		}
		targetNetwork, err = datastore.NetworkByID(networkID)
		if err != nil {
			rl.Error(err.Error())
			return context.NoContent(http.StatusInternalServerError)
		}
	default:
		err = errors.New("unknown access type")
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}

	session := api.CreateSessionDTO{
		OriginIP:              context.RealIP(),
		UserName:              userName,
		TargetProtocolID:      targetProtocolID,
		TargetHost:            params.Get("hostname"),
		TargetPort:            targetPort,
		AccessType:            accessType,
		MandateID:             mandateID,
		CustomTargetNetworkID: networkID,
		CustomTargetLogin:     params.Get("custom_login"),
		CustomTargetPassword:  params.Get("custom_password"),
		CustomTargetPrivKey:   params.Get("custom_key"),
	}

	session.CustomTargetPrivKey = normalizeKey(session.CustomTargetPrivKey)
	sessionToken := uuid.NewV4().String()
	err = datastore.CreateSession(sessionToken, session)
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}

	return context.JSON(http.StatusOK, api.SessionLocatorDTO{
		Token:        sessionToken,
		NetworkName:  targetNetwork.Name,
		Endpoint:     targetNetwork.Endpoint,
		Servicepoint: targetNetwork.Servicepoint,
	})
}

func (app *BastionServer) readSessionHandler(context echo.Context) error {
	rl := context.Get(requestLoggerContextKey).(*zap.Logger)
	token := context.Param("token")
	session, err := datastore.Session(token)
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}
	err = context.JSON(http.StatusOK, session)
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}
	err = datastore.DeleteSession(token) // Одноразовый доступ к сессии по токену
	if err != nil {
		rl.Error(err.Error())
		// Удалить хоть и не получилось, но ответ мы сформировали и нужно его отправить, поэтому не возвращаемся с ошибкой
	}
	return nil
}

func (app *BastionServer) createSessionTemplateHandler(context echo.Context) error {
	rl := context.Get(requestLoggerContextKey).(*zap.Logger)
	userName, ok := context.Get("SID").(string)
	if !ok {
		rl.Error("unable to get SID from request context")
		return context.NoContent(http.StatusInternalServerError)
	}
	params, err := context.FormParams()
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}
	for param := range params {
		rl.Debug("CreateSessionTemplate data dump", zap.String("key", param), zap.String("value", params.Get(param)))
	}

	targetProtocolID, err := strconv.Atoi(params.Get("protocol_id"))
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}
	targetPort, err := strconv.Atoi(params.Get("port"))
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}

	var mandateID int
	var networkID int
	accessType := api.AccessType(params.Get("access_type"))
	switch accessType {
	case api.AccessTypeMandate:
		mandateID, err = strconv.Atoi(params.Get("mandate_id"))
		if err != nil {
			rl.Error(err.Error())
			return context.NoContent(http.StatusInternalServerError)
		}
		err = checkMandate(userName, mandateID)
		if err != nil {
			rl.Error(err.Error())
			return context.NoContent(http.StatusInternalServerError)
		}
	case api.AccessTypeCustom:
		networkID, err = strconv.Atoi(params.Get("custom_network_id"))
		if err != nil {
			rl.Error(err.Error())
			return context.NoContent(http.StatusInternalServerError)
		}
	default:
		err = errors.New("unknown access type")
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}

	st := api.SessionTemplate{
		Name:                  params.Get("name"),
		TargetProtocolID:      targetProtocolID,
		TargetHost:            params.Get("hostname"),
		TargetPort:            targetPort,
		AccessType:            accessType,
		MandateID:             mandateID,
		CustomTargetNetworkID: networkID,
		CustomTargetLogin:     params.Get("custom_login"),
		CustomTargetPassword:  params.Get("custom_password"),
		CustomTargetPrivKey:   params.Get("custom_key"),
	}

	err = datastore.CreateSessionTemplate(userName, st)
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}
	return context.NoContent(http.StatusOK)
}

func (app *BastionServer) deleteSessionTemplateHandler(context echo.Context) error {
	rl := context.Get(requestLoggerContextKey).(*zap.Logger)
	userName, ok := context.Get("SID").(string)
	if !ok {
		rl.Error("unable to get SID from request context")
		return context.NoContent(http.StatusInternalServerError)
	}
	id, err := strconv.Atoi(context.Param("id"))
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}
	err = datastore.DeleteSessionTemplate(userName, id)
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}
	return context.NoContent(http.StatusOK)
}

func (app *BastionServer) readUserData(context echo.Context) error {
	rl := context.Get(requestLoggerContextKey).(*zap.Logger)
	userName, ok := context.Get("SID").(string)
	if !ok {
		rl.Error("unable to get SID from request context")
		return context.NoContent(http.StatusInternalServerError)
	}
	var err error
	data := api.ReadUserDTO{}
	data.User, err = datastore.User(userName)
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}
	data.Mandates, err = datastore.Mandates(userName)
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}
	data.SessionTemplates, err = datastore.SessionTemlates(userName)
	if err != nil {
		rl.Error(err.Error())
		return context.NoContent(http.StatusInternalServerError)
	}
	return context.JSON(http.StatusOK, data)
}

func checkMandate(userName string, givenMandateID int) error {
	userAssignedMandates, err := datastore.Mandates(userName)
	if err != nil {
		return err
	}
	for _, assignedMandate := range userAssignedMandates {
		if assignedMandate.ID == givenMandateID {
			return nil
		}
	}
	return errors.New("user has no given mandate")
}

// normalizeKey нормализует строку, содержащую ключ в формате PEM
// * Убирает переносы строки ('\n') в теле ключа
// * Убирает пробелы в теле ключа
func normalizeKey(key string) string {
	flattenKey := strings.ReplaceAll(key, "\n", "")
	pemEncodedKeyCapturer := regexp.MustCompile("(-{5}[A-Z ]+-{5})(.*)(-{5}[A-Z ]+-{5})")

	s := pemEncodedKeyCapturer.FindAllStringSubmatch(flattenKey, -1)
	if s != nil {
		n := s[0][1] + "\n" + strings.ReplaceAll(s[0][2], " ", "") + "\n" + s[0][3]
		return n
	}
	return ""
}
