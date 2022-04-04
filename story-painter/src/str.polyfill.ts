
/**
 * String.prototype.replaceAll() polyfill
 * https://gomakethings.com/how-to-replace-a-section-of-a-string-with-another-one-with-vanilla-js/
 * @author Chris Ferdinandi
 * @license MIT
 */
if (!String.prototype.replaceAll) {
  (String.prototype as any).replaceAll = function (str: string, newStr: string) {

    // If a regex pattern
    if (Object.prototype.toString.call(str).toLowerCase() === '[object regexp]') {
      return this.replace(str, newStr);
    }

    // If a string
    return this.replace(new RegExp(str, 'g'), newStr);
  };
}


if (!String.prototype.matchAll) {
  (String.prototype as any).matchAll = function (rx: any) {
    if (typeof rx === "string") rx = new RegExp(rx, "g"); // coerce a string to be a global regex
    rx = new RegExp(rx); // Clone the regex so we don't update the last index on the regex they pass us
    let cap = []; // the single capture
    let all = []; // all the captures (return this)
    while ((cap = rx.exec(this)) !== null) all.push(cap); // execute and add
    return all; // profit!
  };
}
