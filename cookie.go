package main

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type cookieName string

const (
	MESSAGE cookieName = "message"
)

func SetCookie(c echo.Context, cookieName cookieName, values string, expires time.Time) {

	cookie := new(http.Cookie)
	cookie.Name = string(cookieName)
	cookie.Value = values
	cookie.Expires = expires
	c.SetCookie(cookie)
}

func GetCookie(c echo.Context, cookieName cookieName) string {
	cookie, err := c.Cookie(string(cookieName))
	if err != nil {
		return ""
	}
	return cookie.Value
}

func ClearCookie(c echo.Context, cookieName cookieName) {
	// Clear the cookie
	cookie := new(http.Cookie)
	cookie.Name = string(cookieName)
	cookie.Value = ""
	cookie.Expires = time.Unix(0, 0)
	c.SetCookie(cookie)
}
