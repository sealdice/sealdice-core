// import * as monaco from 'monaco-editor';
// import type { CharItem } from '~/store';


// // Register a new language
// monaco.languages.register({ id: 'trpg' });

// // Register a tokens provider for the language
// monaco.languages.setMonarchTokensProvider('trpg', {
//   tokenizer: {
//     root: [
//       // 注: 分组不能嵌套，且全部文本必须在分组中
//       [/^([^(]+)(\(\d+\))(\s+)(\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2})$/, [{ token: 'nickname-$1' }, { token: 'userid' }, { token: '' }, { token: 'time', next: '@speak.$1' }]],
//     ],
//     speak: [
//       [/^([^(]+)(\(\d+\))(\s+)(\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2})$/, { token: '@rematch' }, '@pop'],
//       [/.+/, { token: 'message-$S2' }]
//     ]
//   }
// });


// // Define a new theme that contains only rules that match this language
// monaco.editor.defineTheme('myCoolTheme', {
//   base: 'vs',
//   inherit: false,
//   rules: [
//     { token: 'message-木落', foreground: 'f99252' },
//     { token: 'message-海豹一号机', foreground: '008800' },

//     { token: 'nickname', foreground: 'ff0000' },
//   ],
//   colors: {
//     'editor.foreground': '#000000'
//   }
// });


// export function resetTheme(pcList: CharItem[]) {
//   let lst = []
//   for (let i of pcList) {
//     // F = FontStyle (4 bits): None = 0, Italic = 1, Bold = 2, Underline = 4, Strikethrough = 8.
//     if (i.role === '隐藏') {
//       lst.push({ token: `message-${i.name}`, foreground: '#7B7A7A', fontStyle: 'strikethrough' })
//       lst.push({ token: `nickname-${i.name}`, foreground: '#7B7A7A', fontStyle: 'strikethrough' })
//     } else {
//       lst.push({ token: `message-${i.name}`, foreground: `${i.color}` })
//     }
//   }

//   monaco.editor.defineTheme('myCoolTheme', {
//     base: 'vs',
//     inherit: false,
//     rules: [
//       ...lst,  
//       { token: 'nickname', foreground: 'ff0000' },
//     ],
//     colors: {
//       'editor.foreground': '#000000'
//     }
//   }); 
// }
