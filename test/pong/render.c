#include <GLUT/glut.h>
#include <OpenGL/gl.h>
#include "paddle.h"
#include "ball.h"

extern float playerPaddleY; // Declare the external playerPaddleY variable
extern float computerPaddleY; // Declare the external computerPaddleY variable
extern float ballX; // Declare the external ballX variable
extern float ballY; // Declare the external ballY variable

void renderPaddle() {
    // Render player paddle
    glRectf(50.0f, playerPaddleY - 50.0f, 70.0f, playerPaddleY + 50.0f);
    // Render computer paddle
    glRectf(730.0f, computerPaddleY - 50.0f, 750.0f, computerPaddleY + 50.0f);
}

void renderBall() {
    // Render the ball
    glRectf(ballX - 10.0f, ballY - 10.0f, ballX + 10.0f, ballY + 10.0f);
}