// axios 모듈 불러오기
const axios = require('axios');

// GET 요청 예제
axios.get('/proxy/ifconfig.me/get/')
    .then(response => {
        console.log('GET Response:@@@@', response.data);
    })
    .catch(error => {
        console.error('Error in GET request:', error);
    });

// // POST 요청 예제
// axios.post('http://ifconfig.me', )
//   .then(response => {
//     console.log('POST Response:', response.data);
//   })
//   .catch(error => {
//     console.error('Error in POST request:', error);

//   });
