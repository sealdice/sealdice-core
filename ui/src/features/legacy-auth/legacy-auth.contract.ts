import { getLegacySalt, legacySignin } from './legacyApi';
import { passwordHash, sleep } from './crypto';
import { useLegacyAuthSession } from './useLegacyAuthSession';

async function assertLegacyAuthContract(): Promise<void> {
  const saltResult = await getLegacySalt();
  const salt: string = saltResult.salt;
  const hashedPassword: string = await passwordHash(salt, 'password');

  await sleep(1);

  const signinResult = await legacySignin(hashedPassword);
  const token: string = signinResult.token;

  const session = useLegacyAuthSession();
  const hasToken: boolean = session.hasLegacyAccessToken.value;
  const loadedSalt: string = session.salt.value;
  const autoSignedIn: boolean = await session.tryAutoSignin();
  const tokenFromPassword: string = await session.signinWithPassword('password');
  const tokenFromHash: string = await session.signinWithHash(hashedPassword);

  session.clearLegacySession();

  void salt;
  void token;
  void hasToken;
  void loadedSalt;
  void autoSignedIn;
  void tokenFromPassword;
  void tokenFromHash;
}

void assertLegacyAuthContract;
