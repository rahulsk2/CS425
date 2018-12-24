import string
import random

def generateAlphabets(filename):
    with open(filename, 'w', encoding='utf-8') as f:
        for i in range(100000):
            for j in string.ascii_uppercase:
                print(j, file=f)


def generateFruits(filename):
    fruits = ["mango", "orange", "apple", "banana"]
    with open(filename, 'w', encoding='utf-8') as f:
        for i in range(400000):
            index = random.randint(0, len(fruits)-1)
            print(fruits[index], file=f)


def generateColors(filename):
    colors = ["red", "yellow", "black", "green"]
    with open(filename, 'w', encoding='utf-8') as f:
        for i in range(500000):
            index = random.randint(0, len(colors)-1)
            print(colors[index], file=f)

def generateTextFile(inputFile, outputFile, noOfCopies):
    with open(inputFile, "r", encoding='utf-8') as f:
        fileContent = f.readlines()
    with open(outputFile, "w", encoding='utf-8') as f:
        for i in range(noOfCopies):
            for each_line in fileContent:
                print(each_line, file=f)


if __name__ == "__main__":
    generateAlphabets("Alphabets_50k")
    generateFruits('Fruits_100k')
    generateColors("Colors_200k")
    # generateTextFile("InputUIUC.txt", "InputTextData", 500)