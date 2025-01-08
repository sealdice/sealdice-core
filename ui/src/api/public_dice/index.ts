import request from '..';
export function getDicePublicInfo() {
  return request<DicePublicInfo>('get', 'dice/public/info');
}
export function setDicePublicInfo(config: any[], selected: any[]) {
  return request<{ result: true } | { result: false; err: string }>(
    'post',
    'dice/public/set',
    { config: config, selected: selected },
    'json',
  );
}
type DicePublicInfo = {
  endpoints: any[];
  config: any[];
};
