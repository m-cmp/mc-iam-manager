package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/m-cmp/mc-iam-manager/model"
	"github.com/m-cmp/mc-iam-manager/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetUsers returns all users
// @Security BearerAuth
func (h *UserHandler) GetUsers(c echo.Context) error {
	users, err := h.userService.GetUsers(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "사용자 목록을 가져오는데 실패했습니다"})
	}

	return c.JSON(http.StatusOK, users)
}

// GetUser returns a user by ID
// @Security BearerAuth
func (h *UserHandler) GetUserByID(c echo.Context) error {
	id := c.Param("id")
	user, err := h.userService.GetUserByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "사용자를 찾을 수 없습니다"})
	}

	return c.JSON(http.StatusOK, user)
}

// GetUserByUsername returns a user by username
// @Security BearerAuth
func (h *UserHandler) GetUserByUsername(c echo.Context) error {
	username := c.Param("username")
	user, err := h.userService.GetUserByUsername(c.Request().Context(), username)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "사용자를 찾을 수 없습니다"})
	}

	return c.JSON(http.StatusOK, user)
}

// CreateUser creates a new user
// @Security BearerAuth
func (h *UserHandler) CreateUser(c echo.Context) error {
	var user model.User
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "잘못된 요청 형식입니다"})
	}

	if err := h.userService.CreateUser(c.Request().Context(), &user); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "사용자 생성에 실패했습니다"})
	}

	return c.JSON(http.StatusCreated, user)
}

// UpdateUser updates a user
// @Security BearerAuth
func (h *UserHandler) UpdateUser(c echo.Context) error {
	id := c.Param("id")
	var user model.User
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "잘못된 요청입니다",
		})
	}

	user.ID = id
	if err := h.userService.UpdateUser(c.Request().Context(), &user); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "사용자 수정에 실패했습니다",
		})
	}

	return c.JSON(http.StatusOK, user)
}

// DeleteUser deletes a user
// @Security BearerAuth
func (h *UserHandler) DeleteUser(c echo.Context) error {
	id := c.Param("id")
	if err := h.userService.DeleteUser(c.Request().Context(), id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "사용자 삭제에 실패했습니다",
		})
	}

	return c.NoContent(http.StatusNoContent)
}
