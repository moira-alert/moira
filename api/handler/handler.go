package handler

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var database moira.Database

func NewHandler(db moira.Database) http.Handler {
	database = db
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.NoCache) //todo неадекватно много всего проставляет, разобраться
	router.Use(render.SetContentType(render.ContentTypeJSON))

	router.Route("/api", func(router chi.Router) {
		router.Route("/user", user())
		router.Route("/trigger", triggers())
		router.Route("/tag", tag())
		router.Route("/pattern", pattern())
		router.Route("/event", event())
		router.Route("/contact", contact())
		router.Route("/subscription", subscription())
		router.Route("/notification", notification())
	})
	return router
}

func notification() func(router chi.Router) {
	return func(router chi.Router) {
		router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
			//todo хрен знает, что делает, очень похоже на то, что получает нотификаю, которую уже нужно отправить
		})

		router.Delete("/", func(writer http.ResponseWriter, request *http.Request) {
			//todo хрен знает, что делает, очень похоже на то, что удаляет нотификаю, которую уже нужно отправить
		})
	}
}

func subscription() func(router chi.Router) {
	return func(router chi.Router) {
		router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
			//Взять все подписки пользователя
		})

		router.Put("/", func(writer http.ResponseWriter, request *http.Request) {
			//todo какой-то check_json
			//Создать новую подписку
		})

		router.Route("/{SubscriptionId}", func(router chi.Router) {
			router.Delete("/", func(writer http.ResponseWriter, request *http.Request) {
				//Удалить подписку
			})

			router.Put("/test", func(writer http.ResponseWriter, request *http.Request) {
				//Выполнятеся после сохранения, высылает нотификацию сразу
			})
		})
	}
}

func contact() func(router chi.Router) {
	return func(router chi.Router) {
		router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
			//Дергает абсолютно все контакты
		})

		router.Put("/", func(writer http.ResponseWriter, request *http.Request) {
			//todo какой-то check_json
			//Создает новый контакт в админке пользователя
		})

		router.Delete("/{ContactId}", func(writer http.ResponseWriter, request *http.Request) {
			//удалить контакт
		})
	}
}

func event() func(router chi.Router) {
	return func(router chi.Router) {
		router.Get("/{triggerId}", func(writer http.ResponseWriter, request *http.Request) {
			//?p=0&size=100
			//взять эвенты по треггеру
		})
	}
}

func pattern() func(router chi.Router) {
	return func(router chi.Router) {
		router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
			//Вытащить все паттерны по всем метрикам
		})

		router.Delete("/{pattern}", func(writer http.ResponseWriter, request *http.Request) {
			//удалить паттерн
			//todo не используется
		})
	}
}

func tag() func(router chi.Router) {
	return func(router chi.Router) {
		router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
			tagsNames, err := database.GetTagNames()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err.Error())
				http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			tagsMap, err := database.GetTags(tagsNames)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err.Error())
				http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			tagsData := TagsData{
				TagNames: tagsNames,
				TagsMap:  tagsMap,
			}

			if err := render.Render(writer, request, &tagsData); err != nil {
				return
			}
		})

		router.Get("/stats", func(writer http.ResponseWriter, request *http.Request) {
			//вытащить все подписки по всем тегам
			//todo не используется
		})

		router.Route("/{tag}", func(router chi.Router) {
			router.Delete("/", func(writer http.ResponseWriter, request *http.Request) {
				//удалить триггер к хуям
				//todo не используется
			})

			router.Put("/data", func(writer http.ResponseWriter, request *http.Request) {
				//todo какой-то check_json
				//Постим майтейнс для тега
			})
		})
	}
}

type TagsData struct {
	TagNames []string                 `json:"list"`
	TagsMap  map[string]moira.TagData `json:"tags"`
}

func (*TagsData) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func user() func(router chi.Router) {
	return func(router chi.Router) {
		router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
			if err := render.Render(writer, request, &User{request.Header.Get("login")}); err != nil {
				return
			}
		})

		router.Get("/settings", func(writer http.ResponseWriter, request *http.Request) {
			//todo не забыть пропихнуть user в каждый subscription

			userLogin := request.Header.Get("login")
			fmt.Fprintf(os.Stderr, "%s", userLogin)
			userSettings := UserSettings{
				User:          User{userLogin},
				Contacts:      make([]moira.ContactData, 0),
				Subscriptions: make([]moira.SubscriptionData, 0),
			}

			subscriptionIds, err := database.GetUserSubscriptions(userLogin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err.Error())
				http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			contactIds, err := database.GetUserContacts(userLogin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s", err.Error())
				http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			for _, id := range subscriptionIds {
				subscription, err := database.GetSubscription(id)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s", err.Error())
					http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
				userSettings.Subscriptions = append(userSettings.Subscriptions, subscription)
			}

			for _, id := range contactIds {
				contact, err := database.GetContact(id)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s", err.Error())
					http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
				userSettings.Contacts = append(userSettings.Contacts, contact)
			}
			if err := render.Render(writer, request, &userSettings); err != nil {
				return
			}
		})
	}
}

type UserSettings struct {
	User
	Contacts      []moira.ContactData      `json:"contacts"`
	Subscriptions []moira.SubscriptionData `json:"subscriptions"`
}

func (*UserSettings) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

type User struct {
	Login string `json:"login"`
}

func (*User) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func triggers() func(router chi.Router) {
	return func(router chi.Router) {
		router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
			//todo очень странная параша, отдает все триггреры
		})

		router.Put("/", func(writer http.ResponseWriter, request *http.Request) {
			//Сохранение триггреа при его создании
		})

		router.Get("/page", func(writer http.ResponseWriter, request *http.Request) {
			var onlyErrors bool
			var filterTags []string

			filterOkCookie, err := request.Cookie("moira_filter_ok")
			if err == http.ErrNoCookie {
				onlyErrors = false
			} else {
				onlyErrors = filterOkCookie.Value == "true"
			}

			fillerTagsCookie, err := request.Cookie("moira_filter_tags")
			if err == http.ErrNoCookie {
				filterTags = make([]string, 0)
			} else {
				str := strings.Split(fillerTagsCookie.Value, ",")
				for _, tag := range str {
					if tag != "" {
						filterTags = append(filterTags, tag)
					}
				}
			}

			page, err := strconv.ParseInt(request.URL.Query().Get("p"), 10, 64)
			if err != nil {
				page = 0
			}
			size, err := strconv.ParseInt(request.URL.Query().Get("size"), 10, 64)
			if err != nil {
				size = 10
			}
			fmt.Fprintln(os.Stderr, onlyErrors, len(filterTags))

			var triggersChecks []moira.TriggerChecksData
			var total int64
			var triggerIds []string

			if !onlyErrors && len(filterTags) == 0 {
				triggerIds, total, err = database.GetTriggerIds()
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s", err.Error())
					http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
				triggerIds = triggerIds[page*size : (page+1)*size]
				triggersChecks, err = database.GetTriggersChecks(triggerIds)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s", err.Error())
					http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}
			} else {
				triggerIds, total, err = database.GetFilteredTriggersIds(filterTags, onlyErrors)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s", err.Error())
					http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}

				from := page * size
				to := (page + 1) * size

				if from > total {
					from = total
				}

				if to > total {
					to = total
				}

				triggerIds = triggerIds[from:to]
				triggersChecks, err = database.GetTriggersChecks(triggerIds)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s", err.Error())
					http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}

			}

			triggersList := TriggersList{
				List:  triggersChecks,
				Total: total,
				Page:  page,
				Size:  size,
			}
			//todo Выпилить лишние поля из JSON'a

			if err := render.Render(writer, request, &triggersList); err != nil {
				return
			}
		})
		router.Route("/{triggerId}", trigger())
	}
}

type TriggersList struct {
	Page  int64                     `json:"page"`
	Size  int64                     `json:"size"`
	Total int64                     `json:"total"`
	List  []moira.TriggerChecksData `json:"list"`
}

func (*TriggersList) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func trigger() func(router chi.Router) {
	return func(router chi.Router) {
		router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
			//дать триггер
		})

		router.Get("/state", func(writer http.ResponseWriter, request *http.Request) {
			//дать состояние триггера
		})

		router.Route("/throttling", func(router chi.Router) {
			router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
				//not found
			})

			router.Delete("/", func(writer http.ResponseWriter, request *http.Request) {
				//удалить throttling
			})
		})

		router.Route("/metrics", func(router chi.Router) {
			router.Get("/", func(writer http.ResponseWriter, request *http.Request) {
				//not found
			})

			router.Delete("/", func(writer http.ResponseWriter, request *http.Request) {
				//?name=EG-FRONT-03.UserSettings.Read
			})
		})

		router.Put("/maintenance", func(writer http.ResponseWriter, request *http.Request) {
			//Установить maintenance
			//в body - время, до которого будет maintenance
			//Умеет в массив треггеров
		})
	}
}
