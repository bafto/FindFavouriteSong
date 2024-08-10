#include "DDP/ddpmemory.h"
#include "DDP/ddptypes.h"
#include <stdio.h>
#include <string.h>

#define CURL_STATICLIB
#include "curl/curl.h"

typedef ddpint pointer;
typedef CURL *CurlClient;

void Setzte_URL(CurlClient curl, ddpstring *url) {
	curl_easy_setopt(curl, CURLOPT_URL, url->str);
}

void Setze_POST_Daten(CurlClient curl, ddpstringref data) {
	curl_easy_setopt(curl, CURLOPT_POSTFIELDS, data->str);
}

void Setze_Methode(CurlClient curl, ddpstring *method) {
	curl_easy_setopt(curl, CURLOPT_CUSTOMREQUEST, method->str);
}

void Setze_Curl_Zertifikat(CurlClient curl, ddpstring *zertifikat) {
	curl_easy_setopt(curl, CURLOPT_CAINFO, zertifikat->str);
}

pointer Setze_Kopfzeile(CurlClient curl, ddpstringlistref headers) {
	struct curl_slist *list = NULL;
	for (ddpint i = 0; i < headers->len; i++) {
		if (ddp_strlen(&headers->arr[i]) > 0) {
			list = curl_slist_append(list, headers->arr[i].str);
		}
	}
	curl_easy_setopt(curl, CURLOPT_HTTPHEADER, list);
	return (pointer)list;
}

void Befreie_Kopfzeile(struct curl_slist *list) {
	curl_slist_free_all(list);
}

extern ddpint DDP_Write_Data(pointer, ddpint, ddpint, ddpstringref);

void Setzte_Curl_Koerper_Ziel(CurlClient curl, ddpstringref ziel) {
	curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, DDP_Write_Data);
	curl_easy_setopt(curl, CURLOPT_WRITEDATA, ziel);
}

void Curl_Fehler(ddpstring *ret, CURLcode code) {
	const char *msg = curl_easy_strerror(code);
	int len = strlen(msg);
	ret->str = DDP_ALLOCATE(char, len + 1);
	ret->cap = len + 1;
	memcpy(ret->str, msg, ret->cap);
}

ddpint Curl_Status(CURL *curl) {
	long dest;
	curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &dest);
	return (ddpint)dest;
}
