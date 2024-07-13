#include <GLUT/glut.h>
#include <OpenGL/gl.h>
#include <stdbool.h>
#include "paddle.h"
#include "ball.h"
#include "render.h"

bool keyStates[256] = { false }; // Array to hold the state of keys
bool specialKeyStates[256] = { false }; // Array to hold the state of special keys

void keyPressed(unsigned char key, int x, int y) {
    keyStates[key] = true; // Set the state of the current key to pressed
}

void keyReleased(unsigned char key, int x, int y) {
    keyStates[key] = false; // Set the state of the current key to released
}

void specialKeyPressed(int key, int x, int y) {
    specialKeyStates[key] = true; // Set the state of the current special key to pressed
}

void specialKeyReleased(int key, int x, int y) {
    specialKeyStates[key] = false; // Set the state of the current special key to released
}

void init() {
    // Initialize game objects
    initPaddle();
    initBall();
}

void display() {
    glClear(GL_COLOR_BUFFER_BIT);
    renderPaddle();
    renderBall();
    glutSwapBuffers();
}

void update(int value) {
    // Update game objects
    updatePaddle();
    updateBall();
    glutPostRedisplay();
    glutTimerFunc(16, update, 0); // 60 FPS
}

int main(int argc, char** argv) {
    glutInit(&argc, argv);
    glutInitDisplayMode(GLUT_DOUBLE | GLUT_RGB);
    glutInitWindowSize(800, 600);
    glutCreateWindow("Pong Game");

    // Set up the OpenGL context
    glMatrixMode(GL_PROJECTION);
    glLoadIdentity();
    gluOrtho2D(0.0, 800.0, 0.0, 600.0);
    glMatrixMode(GL_MODELVIEW);

    init();

    glutDisplayFunc(display);
    glutKeyboardFunc(keyPressed);
    glutKeyboardUpFunc(keyReleased);
    glutSpecialFunc(specialKeyPressed);
    glutSpecialUpFunc(specialKeyReleased);
    glutTimerFunc(16, update, 0);

    glutMainLoop();
    return 0;
}