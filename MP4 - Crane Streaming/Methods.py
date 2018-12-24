import sys
import string
#arguments are always in the form of lines

vowels = ('a','e','i','o','u','A','E','I','O','U')



def evenFilter(arg): #takes integers and filters out the odd integers
    arg1=int(arg)
    if arg1%2==0:
        return (True,arg1)
    else:
        return (False,None)



def tripleTransform(arg): #takes integers and multiplies them by 3
    arg1=int(arg)
    return 3*arg1


def alphaToIntTransform(arg):
    int_val=ord(arg)
    return int_val


def excludeVowelsFilter(arg):
    vowels = ('a','e','i','o','u','A','E','I','O','U')
    if arg.startswith(vowels):
        return (False, None)
    else:
        return (True, arg)


def execFilter(funchandle):
    line=input()
    while line!='TOFFEE':
        retval=funchandle(line)
        if retval[0]==True:
            print(retval[1])
        line=input()
    print('TOFFEE')


def execTransform(funchandle):
    line=input()
    while line!='TOFFEE':
        retval=funchandle(line)
        print(retval)
        line=input()
    print('TOFFEE')


def spout(fname,skip):
    with open(fname,'r') as f:
        for i in range(skip):
            next(f)
        for line in f:
            yield line


def execSpout(fname,skip):
    tap = spout(fname,skip)
    for line in tap:
        print(line,end='')
    print('TOFFEE')


def JoinUp(line,database):
    for entry in database:
        #place predicate here to test for
        if line==entry[:-1]:
            yield line+','+entry


def execJoin(funchandle,fname):
    with open(fname,'r') as f:
        database=f.readlines()
    line=input()
    while line!='TOFFEE':
        joiner=funchandle(line,database)
        for elem in joiner:
            print(elem)
        line=input()
    print('TOFFEE')


def JoinIfDifferent(line, database):
    result = []
    for each_line in database:
        if line.lower() != each_line[:-1].lower():
            result.append(line + "," + each_line[:-1])
    return result


def AlphaJoin(fname):
    with open(fname,'r') as f:
        database=f.readlines()
    line=input()
    while line!='TOFFEE':
        joiner=JoinIfDifferent(line, database)
        for elem in joiner:
            print(elem)
        line=input()
    print('TOFFEE')



# counter=0
def wcTransform():
    line = input()
    counterstrike={}
    counter=0
    while line!='TOFFEE':
        counter+=1
        counterstrike[line] = counterstrike.get(line, 0) + 1
        if counter%10000 == 0:
            print(counterstrike)
        line = input()
    print(counterstrike)
    print("Total Values:", sum(counterstrike.values()))
    print('TOFFEE')


def RemoveGarbageTransform():
    line = input()
    while line!='TOFFEE':
        result = []
        for each_char in list(line):
            if each_char.isalpha():
                result.append(each_char)
        print(''.join(result))
        line = input()
    print('TOFFEE')


def asciiTransform():
    line = input()
    line_counter = 0
    total_sum = 0
    while line!='TOFFEE':
        line_counter += 1
        sum=0
        for each_char in list(line):
            sum += ord(each_char)
        total_sum += sum
        print("Line:", line_counter , " => Coded Value:", sum)
        line = input()
    print("Total Lines:", line_counter , " => Sum of all Coded Values:", total_sum)
    print('TOFFEE')


def excludeVowelsFilter_WithState():
    line = input()
    while line!='TOFFEE':
        if not line.startswith(vowels):
            print(line)
        line=input()
    print('TOFFEE')


def excludeConsonantsFilter():
    line = input()
    while line!='TOFFEE':
        if line.startswith(vowels):
            print(line)
        line=input()
    print('TOFFEE')


if __name__=="__main__":
    #funcmap={0: execSpout,1:evenFilter,2:tripleTransform, 3:alphaToIntTransform, 4:excludeVowelsFilter, 5: JoinUp}
    funcmap={0: execSpout,1:wcTransform, 2:excludeVowelsFilter_WithState, 3:AlphaJoin, 4:excludeConsonantsFilter, 5:RemoveGarbageTransform, 6:asciiTransform}
    funcnum=int(sys.argv[1])
    functype=sys.argv[2]
    if functype=='JOIN':
        fname=sys.argv[3]
        funcmap[funcnum](fname)
    elif functype=='FILTER':
        funcmap[funcnum]()
    elif functype=='TRANSFORM':
        funcmap[funcnum]()
    elif functype=='SPOUT':
        fname=sys.argv[3]
        skip=int(sys.argv[4])
        execSpout(fname,skip)
    # if functype=='JOIN':
    #     fname=sys.argv[3]
    #     execJoin(funcmap[funcnum],fname)
    # elif functype=='FILTER':
    #     execFilter(funcmap[funcnum])
    # elif functype=='TRANSFORM':
    #     execTransform(funcmap[funcnum])
    # elif functype=='SPOUT':
    #     fname=sys.argv[3]
    #     skip=int(sys.argv[4])
    #     execSpout(fname,skip)
