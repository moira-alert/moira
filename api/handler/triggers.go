package handler

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/moira-alert/moira-alert"
	"github.com/moira-alert/moira-alert/api/dto"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func triggers(router chi.Router) {
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

		triggersList := dto.TriggersList{
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
	router.Route("/{triggerId}", trigger)
}
