// Cuckoo Cycle, a memory-hard proof-of-work
// Copyright (c) 2013-2016 John Tromp

#include "cuckoo.h"
#include <inttypes.h> // for SCNx64 macro
#include <stdio.h>    // printf/scanf
#include <stdlib.h>   // exit
#include <unistd.h>   // getopt
#include <assert.h>   // d'uh

// arbitrary length of header hashed into siphash key
#define HEADERLEN 80

char* slurp(char const* path) {
    FILE *fp;
    long length;
    char *buffer;

    fp = fopen(path, "rb");
    if (fp == NULL) {
        return NULL;
    }

    fseek(fp , 0, SEEK_END);
    length = ftell(fp);
    fseek(fp, 0, SEEK_SET);

    buffer = (char*) malloc((length + 1) * sizeof(char));
    if (buffer) {
        fread(buffer, sizeof(char), length, fp);
    }
    fclose(fp);

    buffer[length] = '\0';

    // for (int i = 0; i < length; i++) {
    //     printf("buffer[%d] == %x\n", i, buffer[i]);
    // }

    return buffer;
}

int main(int argc, char **argv) {
  char header[HEADERLEN];
  int nonce = 0;
  unsigned len;
  int c;

  memset(header, 0, sizeof(header));
  while ((c = getopt (argc, argv, "h:n:i:")) != -1) {
    switch (c) {
      case 'h':
        len = strlen(optarg);
        assert(len <= sizeof(header));
        memcpy(header, optarg, len);
        break;
      case 'n':
        nonce = atoi(optarg);
        break;
      case 'i':
        char *data;
        data = slurp(optarg);
        memcpy(header, data, HEADERLEN);
        break;
    }
  }
  char headernonce[HEADERLEN];
  u32 hdrlen = sizeof(header);
  memcpy(headernonce, header, hdrlen);
  memset(headernonce+hdrlen, 0, sizeof(headernonce)-hdrlen);
  ((u32 *)headernonce)[HEADERLEN/sizeof(u32)-1] = htole32(nonce);
  siphash_keys keys;
  setheader(headernonce, sizeof(headernonce), &keys);
  printf("Verifying size %d proof for cuckoo%d(\"%s\",%d)\n",
               PROOFSIZE, EDGEBITS+1, header, nonce);
  for (int nsols=0; scanf(" Solution") == 0; nsols++) {
    word_t nonces[PROOFSIZE];
    for (int n = 0; n < PROOFSIZE; n++) {
      uint64_t nonce;
      int nscan = scanf(" %" SCNx64, &nonce);
      assert(nscan == 1);
      nonces[n] = nonce;
    }
    int pow_rc = verify(nonces, &keys);
    if (pow_rc == POW_OK) {
      printf("Verified with cyclehash ");
      unsigned char cyclehash[32];
      blake2b((void *)cyclehash, sizeof(cyclehash), (const void *)nonces, sizeof(nonces), 0, 0);
      for (int i=0; i<32; i++)
        printf("%02x", cyclehash[i]);
      printf("\n");
    } else {
      printf("FAILED due to %s\n", errstr[pow_rc]);
    }
  }
  return 0;
}
