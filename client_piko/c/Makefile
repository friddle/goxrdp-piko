CC = gcc
CFLAGS = -Wall -Wextra -O2 -fPIC
AR = ar
ARFLAGS = rcs

# 目标文件
OBJS = rle.o
LIB = librle.a

# 默认目标
all: $(LIB)

# 编译静态库
$(LIB): $(OBJS)
	$(AR) $(ARFLAGS) $@ $^

# 编译目标文件
%.o: %.c
	$(CC) $(CFLAGS) -c $< -o $@

# 清理
clean:
	rm -f $(OBJS) $(LIB)

.PHONY: all clean 