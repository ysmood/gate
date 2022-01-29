import http from 'k6/http';
import { sleep } from 'k6';

export default function () {
  http.get('https://test.vane.im:3000');
  sleep(1);
}