#include <GLUT/glut.h>
#include <OpenGL/gl.h>
#include <stdbool.h>
#include "paddle.h"
#include "ball.h"

extern bool keyStates[256]; // Declare the external keyStates array
extern bool specialKeyStates[256]; // Declare the external specialKeyStates array

float playerPaddleY = 300.0f;
float computerPaddleY = 300.0f;
float playerPaddleSpeed = 5.0f;
float computerPaddleSpeed = 3.0f;

void initPaddle() {
    // Initialize paddle positions
    playerPaddleY = 300.0f;
    computerPaddleY = 300.0f;
}

void updatePaddle() {
    // Update player paddle position based on keyboard input
    if (specialKeyStates[GLUT_KEY_UP]) {
        playerPaddleY += playerPaddleSpeed;
    }
    if (specialKeyStates[GLUT_KEY_DOWN]) {
        playerPaddleY -= playerPaddleSpeed;
    }

    // Ensure the paddle stays within the window bounds
    if (playerPaddleY + 50.0f > 600.0f) {
        playerPaddleY = 550.0f;
    }
    if (playerPaddleY - 50.0f < 0.0f) {
        playerPaddleY = 50.0f;
    }

    // Update computer paddle position based on ball position
    if (getBallY() > computerPaddleY) {
        computerPaddleY += computerPaddleSpeed;
    } else if (getBallY() < computerPaddleY) {
        computerPaddleY -= computerPaddleSpeed;
    }

    // Ensure the paddle stays within the window bounds
    if (computerPaddleY + 50.0f > 600.0f) {
        computerPaddleY = 550.0f;
    }
    if (computerPaddleY - 50.0f < 0.0f) {
        computerPaddleY = 50.0f;
    }
}