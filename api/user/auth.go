package user

import (
    "github.com/0xJacky/Nginx-UI/api"
    "github.com/0xJacky/Nginx-UI/model"
    "github.com/0xJacky/Nginx-UI/settings"
    "net/http"

    "github.com/casdoor/casdoor-go-sdk/casdoorsdk"
    "github.com/gin-gonic/gin"
    "github.com/pkg/errors"
    "golang.org/x/crypto/bcrypt"
    "gorm.io/gorm"
)

type LoginUser struct {
    Name     string `json:"name" binding:"required,max=255"`
    Password string `json:"password" binding:"required,max=255"`
}

func Login(c *gin.Context) {
    var user LoginUser
    ok := api.BindAndValid(c, &user)
    if !ok {
        return
    }

    u, _ := model.GetUser(user.Name)

    if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(user.Password)); err != nil {
        c.JSON(http.StatusForbidden, gin.H{
            "message": "The username or password is incorrect",
        })
        return
    }

    token, err := model.GenerateJWT(u.Name)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "message": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "ok",
        "token":   token,
    })
}

func Logout(c *gin.Context) {
    token := c.GetHeader("Authorization")
    if token != "" {
        err := model.DeleteToken(token)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{
                "message": err.Error(),
            })
            return
        }
    }
    c.JSON(http.StatusNoContent, nil)
}

type CasdoorLoginUser struct {
    Code  string `json:"code" binding:"required,max=255"`
    State string `json:"state" binding:"required,max=255"`
}

func CasdoorCallback(c *gin.Context) {
    var loginUser CasdoorLoginUser
    ok := api.BindAndValid(c, &loginUser)
    if !ok {
        return
    }
    endpoint := settings.CasdoorSettings.Endpoint
    clientId := settings.CasdoorSettings.ClientId
    clientSecret := settings.CasdoorSettings.ClientSecret
    certificate := settings.CasdoorSettings.Certificate
    organization := settings.CasdoorSettings.Organization
    application := settings.CasdoorSettings.Application
    if endpoint == "" || clientId == "" || clientSecret == "" || certificate == "" || organization == "" || application == "" {
        c.JSON(http.StatusInternalServerError, gin.H{
            "message": "Casdoor is not configured",
        })
        return
    }
    casdoorsdk.InitConfig(endpoint, clientId, clientSecret, certificate, organization, application)
    token, err := casdoorsdk.GetOAuthToken(loginUser.Code, loginUser.State)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "message": err.Error(),
        })
        return
    }
    claims, err := casdoorsdk.ParseJwtToken(token.AccessToken)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "message": err.Error(),
        })
        return
    }
    u, err := model.GetUser(claims.Name)
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            c.JSON(http.StatusForbidden, gin.H{
                "message": "User not exist",
            })
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{
                "message": err.Error(),
            })
        }
        return
    }

    userToken, err := model.GenerateJWT(u.Name)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "message": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "ok",
        "token":   userToken,
    })
}
