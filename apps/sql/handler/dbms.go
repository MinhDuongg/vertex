package handler

import (
	"errors"
	"fmt"

	containersapi "github.com/vertex-center/vertex/apps/containers/api"
	"github.com/vertex-center/vertex/apps/sql/core/port"
	"github.com/vertex-center/vertex/apps/sql/core/types"
	"github.com/vertex-center/vertex/pkg/log"
	"github.com/vertex-center/vertex/pkg/router"
)

type DBMSHandler struct {
	sqlService port.SqlService
}

func NewDBMSHandler(sqlService port.SqlService) port.DBMSHandler {
	return &DBMSHandler{
		sqlService: sqlService,
	}
}

func (r *DBMSHandler) Get(c *router.Context) {
	uuid, apiError := containersapi.GetContainerUUIDParam(c)
	if apiError != nil {
		c.BadRequest(apiError.RouterError())
		return
	}

	inst, apiError := containersapi.GetContainer(c, uuid)
	if apiError != nil {
		c.AbortWithCode(apiError.HttpCode, apiError.RouterError())
		return
	}

	dbms, err := r.sqlService.Get(inst)
	if err != nil {
		c.NotFound(router.Error{
			Code:           types.ErrCodeSQLDatabaseNotFound,
			PublicMessage:  "SQL Database not found.",
			PrivateMessage: err.Error(),
		})
		return
	}

	c.JSON(dbms)
}

func (r *DBMSHandler) Install(c *router.Context) {
	dbms, err := r.getDBMS(c)
	if err != nil {
		return
	}

	serv, apiError := containersapi.GetService(c, dbms)
	if apiError != nil {
		c.AbortWithCode(apiError.HttpCode, apiError.RouterError())
		return
	}

	inst, apiError := containersapi.InstallService(c, serv.ID)
	if apiError != nil {
		c.AbortWithCode(apiError.HttpCode, apiError.RouterError())
		return
	}

	inst.ContainerSettings.Tags = []string{"Vertex SQL", "Vertex SQL - Postgres Database"}
	apiError = containersapi.PatchContainer(c, inst.UUID, inst.ContainerSettings)
	if apiError != nil {
		c.AbortWithCode(apiError.HttpCode, apiError.RouterError())
		return
	}

	inst.Env, err = r.sqlService.EnvCredentials(inst, "postgres", "postgres")
	if err != nil {
		log.Error(err)
		c.Abort(router.Error{
			Code:           types.ErrCodeFailedToConfigureSQLDatabaseContainer,
			PublicMessage:  fmt.Sprintf("Failed to configure SQL Database '%s'.", serv.Name),
			PrivateMessage: err.Error(),
		})
		return
	}

	apiError = containersapi.PatchContainerEnvironment(c, inst.UUID, inst.Env)
	if apiError != nil {
		c.AbortWithCode(apiError.HttpCode, apiError.RouterError())
		return
	}

	c.JSON(inst)
}

func (r *DBMSHandler) getDBMS(c *router.Context) (string, error) {
	db := c.Param("dbms")
	if db != "postgres" {
		c.NotFound(router.Error{
			Code:           types.ErrCodeSQLDatabaseNotFound,
			PublicMessage:  fmt.Sprintf("SQL DBMS not found: %s.", db),
			PrivateMessage: "This SQL DBMS is not supported.",
		})
		return "", errors.New("DBMS not found")
	}

	return db, nil
}
