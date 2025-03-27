package competitions

import (
	"api/database"
	"api/middleware"
	"api/models"
	"api/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// [POST] GetInputFromCompetition
// @Summary Get input from a competition, and create a new try if needed
// @Description Get input from a competition, and create a new try if needed
// @Tags Competitions
// @Accept json
// @Produce json
// @Param inputRequest body InputRequest true "Input request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /competitions/input [post]
// @Security Bearer
func GetInputFromCompetition(c *gin.Context) {
	// Step 1: Authenticate the user
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	// Step 2: Parse the request body
	var inputRequest InputRequest
	if err := c.ShouldBindJSON(&inputRequest); err != nil {
		respondWithError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Step 3: Get the target Competition
	var competition models.Competition
	err = services.GetAccessibleCompetition(user.ID, inputRequest.CompetitionID, &competition)
	if err != nil {
		respondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	// Step 4: Create a new try if it's needed
	services.TriggerPuzzleFirstTry(competition, inputRequest.PuzzleID, inputRequest.PuzzleIndex, inputRequest.PuzzleDifficulty, user.ID)

	// Step 5: Get the puzzle input
	ctx := c.Request.Context()
	puzzleInput, err := services.GetPuzzleInput(competition.CatalogID, competition.CatalogTheme, inputRequest.PuzzleID, user.ID, ctx)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, puzzleInput)
}

// [POST] AnswerPuzzle
// @Summary Answer a part of a puzzle
// @Description Answer a part of a puzzle
// @Tags Competitions
// @Accept json
// @Produce json
// @Param competitionTry body CompetitionTry true "Competition try"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /competitions/answer_puzzle [post]
// @Security Bearer
func AnswerPuzzle(c *gin.Context) {
	// Step 1: Authenticate the user
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	// Step 2: Parse the request body
	var req CompetitionTry
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, ErrInvalidRequest)
		return
	}

	// Step 3: Validate user access to the competition
	var competition models.Competition
	err = services.GetAccessibleCompetition(user.ID, req.CompetitionID, &competition)
	if err != nil {
		respondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	// Step 4: If the user has not downloaded the input yet, it means that the try does not exist
	if _, err := services.GetPuzzleFirstTry(competition.ID, req.PuzzleId, req.PuzzleIndex, user.ID); err != nil {
		if err.Error() == "record not found" {
			respondWithError(c, http.StatusBadRequest, "Try not found")
			return
		}
		respondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Step 5: Validate the try
	isCorrect, err := services.CheckPuzzleAnswer(competition.CatalogID, competition.CatalogTheme, req.PuzzleId, req.PuzzleStep, user.ID, req.Answer)
	if err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to check solution")
		return
	}

	// Step 6: Handle the try (create or update)
	if (!isCorrect) {
		_, err := services.UpdateTry(competition, req.PuzzleId, req.PuzzleIndex, req.PuzzleStep, user.ID, req.Answer)
		if err != nil {
			respondWithError(c, http.StatusInternalServerError, "Failed to update try")
			return
		}
	} else {
		_, err := services.EndTry(competition, req.PuzzleId, req.PuzzleIndex, req.PuzzleStep, user.ID, req.Answer)
		if err != nil {
			respondWithError(c, http.StatusInternalServerError, "Failed to end try")
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"is_correct": isCorrect})
}

// [GET] GetTriesFromCompetitonPuzzle
// @Summary Get all tries for a specific puzzle in a competition
// @Description Get all tries for a specific puzzle in a competition
// @Tags Competitions
// @Accept json
// @Produce json
// @Param comptition_id path string true "Competition ID"
// @Param puzzle_id path string true "Puzzle ID"
// @Param puzzle_index path string true "Puzzle index"
// @Success 200 {array} models.Try
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /competitions/{comptition_id}/puzzles/{puzzle_id}/{puzzle_index}/tries [get]
// @Security Bearer
func GetTriesFromCompetitonPuzzle(c *gin.Context) {
	competitionID := c.Param("id")
	puzzleID := c.Param("puzzle_id")
	puzzleIndex := c.Param("puzzle_index")

	if competitionID == "" || puzzleID == "" {
		respondWithError(c, http.StatusBadRequest, "Invalid competition or puzzle ID")
		return
	}

	// Step 1: Authenticate the user
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	// Step 2: Get the target Competition
	var competition models.Competition
	err = services.GetAccessibleCompetition(user.ID, competitionID, &competition)
	if err != nil {
		respondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	// Step 3: Get the tries for the puzzle
	var tries []models.Try
	if err := database.DB.Where("competition_id = ? AND user_id = ? AND puzzle_id = ? AND puzzle_index = ?",
		competitionID, user.ID, puzzleID, puzzleIndex).Find(&tries).Error; err != nil {
		respondWithError(c, http.StatusInternalServerError, "Failed to fetch tries")
		return
	}

	c.JSON(http.StatusOK, tries)
}

// [GET] UserHasPermissionToViewPuzzle
// @Summary Check if the user has permission to view a puzzle
// @Description Check if the user has permission to view a puzzle
// @Tags Competitions
// @Accept json
// @Produce json
// @Param competition_id path string true "Competition ID"
// @Param puzzle_index path string true "Puzzle index"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /competitions/{id}/puzzles/{puzzle_index}/permission [get]
// @Security Bearer
func UserHasPermissionToViewPuzzle(c *gin.Context) {
	competitionID := c.Param("id")
	puzzleIndex := c.Param("puzzle_index")

	if competitionID == "" || puzzleIndex == "" {
		respondWithError(c, http.StatusBadRequest, "Invalid competition or puzzle ID")
		return
	}

	// Step 1: Authenticate the user
	user, err := middleware.GetUserFromRequest(c)
	if err != nil {
		return
	}

	// Step 2: Get the target Competition
	var competition models.Competition
	err = services.GetAccessibleCompetition(user.ID, competitionID, &competition)
	if err != nil {
		respondWithError(c, http.StatusForbidden, err.Error())
		return
	}

	// Step 3: Check if the user has permission to view the puzzle
	puzzleIndexInt, err := strconv.Atoi(puzzleIndex)
	if err != nil {
		respondWithError(c, http.StatusBadRequest, "Invalid puzzle index")
		return
	}
	hasPermission := services.UserHasPermissionToViewPuzzle(competition, puzzleIndexInt, user.ID)

	c.JSON(http.StatusOK, gin.H{"has_permission": hasPermission})
}