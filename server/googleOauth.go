package server

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"net/http"
	"os"
	"pastebin/models"
	"pastebin/utils"
	"time"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	oauthConfig  *oauth2.Config
	oidcProvider *oidc.Provider
)

func InitGoogleOAuth() {
	var err error
	oidcProvider, err = oidc.NewProvider(context.Background(), "https://accounts.google.com")
	if err != nil {
		log.Fatalf("Ошибка при создании OIDC провайдера: %v", err)
	}

	//fmt.Println(os.Getenv("GOOGLE_CLIENT_ID"))
	oauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		Endpoint:     google.Endpoint,
	}
}

// после нажатия продолжить через гугл ошибка на самом гугле Missing required parameter: redirect_uri Подробнее об этой ошибке…
// Если вы разработчик этого приложения, изучите подробную информацию об ошибке.
// Ошибка 400: invalid_request
func GoogleLoginHandler(w http.ResponseWriter, r *http.Request) {
	url := oauthConfig.AuthCodeURL("random-state")
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func GoogleCallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	code := r.URL.Query().Get("code")

	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		http.Error(w, "Ошибка обмена кода на токен", http.StatusInternalServerError)
		return
	}

	userInfo, err := oidcProvider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		http.Error(w, "Ошибка получения данных пользователя", http.StatusInternalServerError)
		return
	}

	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := userInfo.Claims(&claims); err != nil {
		http.Error(w, "Ошибка обработки данных пользователя", http.StatusInternalServerError)
		return
	}

	// Проверяем, существует ли пользователь
	user, err := FindUserByEmail(ctx, claims.Email)
	if err != nil {
		// Если пользователя нет, создаем его
		userID, err := CreateUser(ctx, claims.Email, claims.Name, "google")
		if err != nil {
			http.Error(w, "Ошибка создания пользователя", http.StatusInternalServerError)
			return
		}
		objID, err := primitive.ObjectIDFromHex(userID)
		if err != nil {
			http.Error(w, "Ошибка преобразования ID", http.StatusInternalServerError)
			return
		}

		user = models.User{
			ID:    objID,
			Email: claims.Email,
			Name:  claims.Name,
		}

	}

	// Генерация токена
	tokenStr := utils.GenerateToken(user.ID, user.Email)

	// Установка куки
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    tokenStr,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	// Перенаправляем на профиль
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}
func FindUserByEmail(ctx context.Context, email string) (models.User, error) {
	var user models.User
	err := db.Collection("users").FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return models.User{}, err
	}
	return user, nil
}

func CreateUser(ctx context.Context, email, name, provider string) (string, error) {
	user := models.User{
		ID:       primitive.NewObjectID(),
		Email:    email,
		Name:     name,
		Provider: provider,
		Role:     "user",
	}
	res, err := db.Collection("users").InsertOne(ctx, user)
	if err != nil {
		return "", err
	}
	return res.InsertedID.(primitive.ObjectID).Hex(), nil
}
