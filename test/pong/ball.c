#include <GLUT/glut.h>
#include <OpenGL/gl.h>
#include "ball.h"
#include "paddle.h"

extern float playerPaddleY; // Declare the external playerPaddleY variable
extern float computerPaddleY; // Declare the external computerPaddleY variable

float ballX = 400.0f;
float ballY = 300.0f;
float ballSpeedX = 4.0f;
float ballSpeedY = 4.0f;

void initBall() {
    // Initialize ball position and speed
    ballX = 400.0f;
    ballY = 300.0f;
    ballSpeedX = 4.0f;
    ballSpeedY = 4.0f;
}

void updateBall() {
    // Update ball position
    ballX += ballSpeedX;
    ballY += ballSpeedY;

    // Check for collision with top and bottom walls
    if (ballY + 10.0f > 600.0f || ballY - 10.0f < 0.0f) {
        ballSpeedY = -ballSpeedY;
    }

    // Check for collision with player paddle
    if (ballX - 10.0f < 70.0f && ballY > playerPaddleY - 50.0f && ballY < playerPaddleY + 50.0f) {
        ballSpeedX = -ballSpeedX;
    }

    // Check for collision with computer paddle
    if (ballX + 10.0f > 730.0f && ballY > computerPaddleY - 50.0f && ballY < computerPaddleY + 50.0f) {
        ballSpeedX = -ballSpeedX;
    }

    // Check for scoring (ball goes out of bounds)
    if (ballX + 10.0f > 800.0f || ballX - 10.0f < 0.0f) {
        initBall(); // Reset ball position and speed
    }
}



float getBallY() {
    return ballY;
}


