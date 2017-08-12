package main

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"

	"github.com/pquerna/ffjson/ffjson"
	"github.com/valyala/fasthttp"

	"github.com/ei-grad/hlcup/db"
	"github.com/ei-grad/hlcup/models"
)

type Application struct {
	db *db.DB
}

func NewApplication() (app Application) {
	app.db = db.New()
	return
}

func (app Application) requestHandler(ctx *fasthttp.RequestCtx) {

	ctx.SetContentType(strApplicationJSON)

	parts := bytes.SplitN(ctx.Path(), []byte("/"), 4)

	switch string(ctx.Method()) {

	case strGet:

		switch len(parts) {
		case 3:

			var resp []byte
			var id uint32

			entity := string(parts[1])

			id64, err := strconv.ParseUint(string(parts[2]), 10, 32)
			if err != nil {
				// 404 - id is not integer
				ctx.SetStatusCode(http.StatusNotFound)
				return
			}
			id = uint32(id64)

			switch entity {

			case strUsers:
				v := app.db.GetUser(id)
				if !v.Valid {
					// 404 - user with given ID doesn't exist
					ctx.SetStatusCode(http.StatusNotFound)
					return
				}
				resp, err = v.MarshalJSON()

			case strLocations:
				v := app.db.GetLocation(id)
				if !v.Valid {
					// 404 - location with given ID doesn't exist
					ctx.SetStatusCode(http.StatusNotFound)
					return
				}
				resp, err = v.MarshalJSON()

			case strVisits:
				v := app.db.GetVisit(id)
				if !v.Valid {
					// 404 - visit with given ID doesn't exist
					ctx.SetStatusCode(http.StatusNotFound)
					return
				}
				resp, err = v.MarshalJSON()

			}

			if err != nil {
				// v.MarshalJSON() failed, shouldn't happen
				panic(err)
			}

			ctx.Write(resp)

			return

		case 4:

			entity := string(parts[1])

			id64, err := strconv.ParseUint(string(parts[2]), 10, 32)
			if err != nil {
				// 404 - id is not integer
				ctx.SetStatusCode(http.StatusNotFound)
				return
			}
			id := uint32(id64)

			tail := string(parts[3])

			if entity == "users" && tail == "visits" {

				visits := app.db.GetUserVisits(id)
				if visits == nil {
					// 404 - user have no visits
					ctx.SetStatusCode(http.StatusNotFound)
					ctx.Logger().Printf("user have no visits")
					return
				}
				ctx.WriteString(`{"visits":[`)
				tmp, _ := visits[0].MarshalJSON()
				ctx.Write(tmp)
				for _, i := range visits[1:] {
					// TODO: implement /users/<id>/visits filters
					ctx.WriteString(",")
					tmp, _ = i.MarshalJSON()
					ctx.Write(tmp)
				}
				ctx.WriteString("]}")
				return

			} else if entity == "locations" && tail == "avg" {

				marks := app.db.GetLocationMarks(id)
				if marks == nil {
					// 404 - no marks for specified location
					ctx.SetStatusCode(http.StatusNotFound)
					ctx.Logger().Printf("location have no marks")
					return
				}
				var sum, count int
				for _, i := range marks {
					// TODO: implement /locations/<id>/avg filters
					sum = sum + int(i.Mark)
					count = count + 1
				}
				ctx.WriteString(fmt.Sprintf(`{"avg": %.5f}`, float64(sum)/float64(count)))
				return

			}

		}

		ctx.SetStatusCode(http.StatusNotFound)

	case strPost:

		// just {} response for POST requests, and Connection:close, yeah
		ctx.SetConnectionClose()
		ctx.Write([]byte("{}"))

		if len(parts) != 3 {
			ctx.SetStatusCode(http.StatusNotFound)
			return
		}

		entity := string(parts[1])

		body := ctx.PostBody()

		if string(parts[2]) == "new" {
			switch entity {
			case strUsers:
				var v models.User
				if err := ffjson.Unmarshal(body, &v); err != nil {
					ctx.SetStatusCode(http.StatusBadRequest)
					ctx.Logger().Printf(err.Error())
					return
				}
				if err := v.Validate(); err != nil {
					ctx.SetStatusCode(http.StatusBadRequest)
					ctx.Logger().Printf(err.Error())
					return
				}
				// XXX: what if it already exists?
				bodyCopy := make([]byte, len(body))
				copy(bodyCopy, body)
				app.db.AddUser(v)
			case strLocations:
				var v models.Location
				if err := ffjson.Unmarshal(body, &v); err != nil {
					ctx.Logger().Printf(err.Error())
					ctx.SetStatusCode(http.StatusBadRequest)
					return
				}
				if err := v.Validate(); err != nil {
					ctx.Logger().Printf(err.Error())
					ctx.SetStatusCode(http.StatusBadRequest)
					return
				}
				// XXX: what if it already exists?
				bodyCopy := make([]byte, len(body))
				copy(bodyCopy, body)
				app.db.AddLocation(v)
			case strVisits:
				var v models.Visit
				if err := ffjson.Unmarshal(body, &v); err != nil {
					ctx.SetStatusCode(http.StatusBadRequest)
					ctx.Logger().Printf(err.Error())
					return
				}
				if err := v.Validate(); err != nil {
					ctx.SetStatusCode(http.StatusBadRequest)
					ctx.Logger().Printf(err.Error())
					return
				}
				// XXX: what if it already exists?
				if err := app.db.AddVisit(v); err != nil {
					ctx.SetStatusCode(http.StatusBadRequest)
					ctx.Logger().Printf(err.Error())
					return
				}
				bodyCopy := make([]byte, len(body))
				copy(bodyCopy, body)
			default:
				ctx.SetStatusCode(http.StatusNotFound)
			}
		} else {
			// TODO: implement updating
			ctx.SetStatusCode(http.StatusNotFound)
			return
		}

	default:
		ctx.SetStatusCode(http.StatusMethodNotAllowed)
		return
	}

}
