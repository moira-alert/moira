package logging

import "github.com/moira-alert/moira"

// DynamicLogger represents wrapper for log which setup minimum log level when added specified field-value
type DynamicLogger struct {
	moira.Logger
	levelByRuleActivated bool
	rules                map[string]interface{}
	levelForActivation   string
}

func ConfigureDynamicLog(logger moira.Logger, activateRules map[string]interface{}, levelForActivation string) moira.Logger {
	return &DynamicLogger{
		Logger:             logger,
		rules:              activateRules,
		levelForActivation: levelForActivation,
	}
}

func (l *DynamicLogger) String(key, value string) moira.Logger {
	if !l.levelByRuleActivated {
		if l.hasRuleForString(key, value) {
			l.activateLevel()
		}
	}
	return l.Logger.String(key, value)
}

func (l *DynamicLogger) hasRuleForString(key string, value string) bool {
	if v, ok := l.rules[key]; ok {
		if s, ok := v.(string); ok && s == value {
			return true
		}

		if ss, ok := v.([]string); ok {
			for _, s := range ss {
				if s == value {
					return true
				}
			}
		}
	}
	return false
}

func (l *DynamicLogger) Int(key string, value int) moira.Logger {
	if !l.levelByRuleActivated {
		if l.hasRuleForInt(key, value) {
			l.activateLevel()
		}
	}
	return l.Logger.Int(key, value)
}

func (l *DynamicLogger) hasRuleForInt(key string, value int) bool {
	if v, ok := l.rules[key]; ok {
		if s, ok := v.(int); ok && s == value {
			return true
		}

		if ss, ok := v.([]int); ok {
			for _, s := range ss {
				if s == value {
					return true
				}
			}
		}
	}
	return false
}

func (l *DynamicLogger) Fields(fields map[string]interface{}) moira.Logger {
	if !l.levelByRuleActivated {
		for key, value := range fields {
			switch val := value.(type) {
			case string:
				if l.hasRuleForString(key, val) {
					l.activateLevel()
				}
			case int:
				if l.hasRuleForInt(key, val) {
					l.activateLevel()
				}
			default:
			}
			if l.levelByRuleActivated {
				break
			}
		}
	}
	return l.Logger.Fields(fields)
}

func (l *DynamicLogger) Level(s string) (moira.Logger, error) {
	if !l.levelByRuleActivated {
		return l.Logger.Level(s)
	}
	return l, nil
}

func (l *DynamicLogger) activateLevel() {
	_, err := l.Logger.Level(l.levelForActivation)
	if err != nil {
		l.Warning("Can't setup dynamic level, err: ", err.Error())
	}
	l.levelByRuleActivated = true
}
