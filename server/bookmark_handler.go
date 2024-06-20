package server

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/techagentng/citizenx/models"
	"github.com/techagentng/citizenx/server/response"
)

func (s *Server) SaveBookmarkReport() gin.HandlerFunc {
	return func(c *gin.Context) {
		if userI, exists := c.Get("user"); exists {
			if user, ok := userI.(*models.User); ok {
				reportID := strings.TrimSpace(c.Param("reportID"))
				if reportID == "" {
					response.JSON(c, "", http.StatusBadRequest, nil, errors.New("cannot ascert model.user user"))
					return
				}

				ok, err := s.IncidentReportService.CheckReportInBookmarkedReport(user.ID, reportID)
				if err != nil {
					response.JSON(c, "", http.StatusInternalServerError, nil, err)
					return
				}
				if ok {
					response.JSON(c, "", http.StatusBadRequest, nil, errors.New("report already bookmarked by the user"))
					return
				}

				bookmarkID := uuid.New()
				bookmarkIDString := bookmarkID.String()
				bookmarkReport := &models.BookmarkReport{
					ID:       bookmarkIDString,
					UserID:   user.ID,
					ReportID: reportID,
				}
				if err := s.IncidentReportService.SaveBookmarkReport(bookmarkReport); err != nil {
					log.Printf(err.Error())
					response.JSON(c, "", http.StatusInternalServerError, nil, errors.New("unable to save handler report"))
					return
				}
				response.JSON(c, "Saved bookmark Successfully", http.StatusCreated, nil, nil)
				return
			}
		}

		response.JSON(c, "", http.StatusInternalServerError, nil, errors.New("unable to save report"))
	}
}
