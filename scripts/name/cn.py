"""
用于处理 https://github.com/wainshine/Chinese-Names-Corpus 语料
"""

import yaml
import random
from pathlib import Path


items = Path('Chinese_Names_Corpus_Gender（120W）.txt').read_text(encoding='utf-8').split('\n')[4:]
fnames = set(Path('姓氏.txt').read_text(encoding='utf-8').split('\n'))

# max([len(x) for x in fnames])  2

lstF = []
lstM = []
lst = []

for i in items:
    try:
        name, gender = i.split(',')
    except:
        print(i)
        continue
    if name[:1] in fnames:
        name = name[1:]
    elif name[:2] in fnames:
        name = name[2:]
    if gender == '男':
        lstM.append(name)
    elif gender == '女':
        lstF.append(name)
    else:
        lst.append(name)

random.seed(3154)
random.shuffle(lstM)
random.shuffle(lstF)
a = lstM[:10000]
b = lstF[:10000]
open('中文男性.csv', 'w', encoding='utf-8').write('\ufeff男性名\n'+'\n'.join(a))
open('中文女性.csv', 'w', encoding='utf-8').write('\ufeff女性名\n'+'\n'.join(b))
