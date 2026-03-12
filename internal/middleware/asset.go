// Copyright (c) 2026 Aptlogica Technologies Private Limited
// SPDX-License-Identifier: MIT
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package middleware

import (
	"serenibase/internal/utils/response"

	app_errors "serenibase/internal/app-errors"

	"github.com/gin-gonic/gin"
)

func checkFileContraints(c *gin.Context, maxFilesAllowed int, maxFileSize int64) error {
	if err := c.Request.ParseMultipartForm(100 << 20); err != nil {
		return app_errors.MultipleFilesTooLargeError
	}

	form, err := c.MultipartForm()
	if err != nil {
		return app_errors.MultipartFormNotFound
	}

	files, filesErr := form.File["files"]
	if !filesErr || files == nil || len(files) == 0 {
		return app_errors.FileNotFound
	}

	if len(files) > maxFilesAllowed {
		return app_errors.TooManyFilesError
	}

	var hasTooLargeFile bool
	for _, fileHeader := range files {
		if fileHeader.Size > int64(maxFileSize) {
			hasTooLargeFile = true
			break
		}
	}
	if hasTooLargeFile {
		return app_errors.MultipleFilesTooLargeError
	}
	return nil
}

func FileSizeLimitMiddleware() gin.HandlerFunc {
	maxFileSize := int64(10 << 20) // 5MB
	maxFilesAllowed := 5

	return func(c *gin.Context) {
		err := checkFileContraints(c, maxFilesAllowed, maxFileSize)
		if err != nil {
			response.CheckAndSendError(c, err)
			c.Abort()
			return
		}

		c.Next()
	}
}
