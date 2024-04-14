"""
用于处理 https://github.com/willnet/gimei 语料
"""

import yaml
from pathlib import Path
conf = yaml.safe_load(Path('names.yml').read_text(encoding='utf-8'))



open('fm.csv', 'w', encoding='utf-8').write('\ufeff'+'\n'.join(['%s,%s' % (x[0], x[1]) for x in conf['first_name']['male']]))
open('ff.csv', 'w', encoding='utf-8').write('\ufeff'+'\n'.join(['%s,%s' % (x[0], x[1]) for x in conf['first_name']['female']]))

open('lm.csv', 'w', encoding='utf-8').write('\ufeff'+'\n'.join(['%s,%s' % (x[0], x[1]) for x in conf['last_name']]))
