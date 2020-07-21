#include "../elrond/context.h"
#include "../elrond/test_utils.h"

byte contractCode[5000] = {};
byte contractID = 0;
byte newAddress[32] = {};

byte deploymentValue[] = {0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,42};

byte arguments[1] = {0};
int argumentsLengths[1] = {1};

void deployChildContract() {
	getArgument(0, &contractID);
	int loadedLength = storageLoad(&contractID, 1, contractCode);
	int64finish(loadedLength);

	byte arg = 0;
	getArgument(1, &arg);

	arguments[0] = arg;

	int initArgLengths[] = {1};
	int result = createContract(
			deploymentValue,
			contractCode,
			loadedLength,
			newAddress,
			1,
			(byte*)argumentsLengths,
			arguments);

	finishResult(result);
}
