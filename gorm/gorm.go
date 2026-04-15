package gorm

import (
	"errors"
	"log"
	"log/slog"
	"net/http"
	"reflect"

	"gorm.io/gorm"
)

var (
	logger = slog.New(slog.NewJSONHandler(log.Writer(), &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
)

type GormOptions struct {
	Model  any
	Result any

	Preload []string
	Select  string
	Limit   int

	KeyModifiers map[string]string
}

type _gorm struct {
	name string
	db   *gorm.DB

	Options GormOptions
}

func (g *_gorm) SetModifiers(modifiers map[string]string) _gorm {
	newg := *g
	newg.Options.KeyModifiers = modifiers

	return newg
}

func (g *_gorm) SetResult(result any) _gorm {
	newg := *g
	newg.Options.Result = result

	return newg
}

func (g _gorm) Data(w http.ResponseWriter, r *http.Request, m map[string]any) (any, error) {
	if g.db == nil {
		return nil, errors.New("gorm: database connection is nil")
	}
	if err := r.ParseForm(); err != nil {
		return nil, err
	}

	tx := g.db.Model(g.Options.Model)

	for k, v := range r.Form {
		for _, value := range v {
			equality := " = "
			if modifier, ok := g.Options.KeyModifiers[k]; ok {
				equality = modifier
			}
			logger.Debug("gorm: adding query condition", "key", k, "value", value, "modifier", equality)
			tx.Where(k+equality+"?", value)
		}
	}

	for _, preload := range g.Options.Preload {
		tx.Preload(preload)
	}

	if g.Options.Select != "" {
		tx.Select(g.Options.Select)
	}

	if g.Options.Limit > 0 {
		tx.Limit(g.Options.Limit)
	}

	t := reflect.TypeOf(g.Options.Result)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	slicePtr := reflect.New(reflect.SliceOf(t))
	tx = tx.Find(slicePtr.Interface())
	if tx.Error != nil {
		logger.Error("gorm: failed to execute query", "error", tx.Error)
		return nil, tx.Error
	} else if tx.RowsAffected == 0 {
		return nil, errors.New("gorm: no records found")
	}

	logger.Info("gorm: query executed successfully")
	return slicePtr.Interface(), nil
}

func (g _gorm) Name() string {
	return g.name
}

// the GORM bridge is designed to provide a low code solution to retrieving
// database information from request parameters.
func GORM(name string, model any, result any, db *gorm.DB, options GormOptions) _gorm {
	return _gorm{
		name: name,
		db:   db,

		Options: GormOptions{
			Model:        model,
			Result:       result,
			Preload:      options.Preload,
			Select:       options.Select,
			Limit:        options.Limit,
			KeyModifiers: options.KeyModifiers,
		},
	}
}
