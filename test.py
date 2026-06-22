def fibonacci(n):
    """
    生成斐波那契数列的函数
    :param n: 生成斐波那契数列的前n项
    :return: 斐波那契数列列表
    """
    fib_sequence = []
    a, b = 0, 1
    for _ in range(n):
        fib_sequence.append(a)
        a, b = b, a + b
    return fib_sequence

# 示例：生成前10项斐波那契数列
if __name__ == "__main__":
    print(fibonacci(10))