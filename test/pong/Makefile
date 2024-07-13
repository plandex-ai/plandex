CC = gcc
CFLAGS = -Wall -g
LDFLAGS = -framework OpenGL -framework GLUT

SRCS = main.c paddle.c ball.c render.c
OBJS = $(SRCS:.c=.o)

all: pong

pong: $(OBJS)
	$(CC) $(CFLAGS) -o pong $(OBJS) $(LDFLAGS)

clean:
	rm -f pong $(OBJS)
