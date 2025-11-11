package logging

func (l *Logger) WithPrefix(prefix string) *Logger {
	newPrefix := prefix
	if l.extraPrefix != "" {
		newPrefix = l.extraPrefix + " " + prefix
	}

	return &Logger{
		appName:     l.appName,
		writer:      l.writer,
		extraPrefix: newPrefix,
	}
}
